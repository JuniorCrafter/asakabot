package main

import (
	"log"
	"strconv"

	"asakabot/internal/broker"
	"asakabot/internal/config"
	"asakabot/internal/handlers"
	"asakabot/internal/i18n"
	"asakabot/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Глобальные переменные для новостей
var lastChannelID int64
var lastMessageID int

func main() {
	cfg := config.LoadConfig()

	db := repository.ConnectDB(cfg.DBUrl)
	defer db.Close()

	rabbitConn := broker.ConnectRabbitMQ(cfg.RabbitMQUrl)
	defer rabbitConn.Close()

	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Panic("Ошибка при подключении к Telegram: ", err)
	}

	log.Printf("Бот @%s запущен!", bot.Self.UserName)

	// Запускаем слушателей в фоне
	go broker.StartConsumer(rabbitConn, "1", bot, db)
	go broker.StartConsumer(rabbitConn, "2", bot, db)
	go broker.StartConsumer(rabbitConn, "3", bot, db)
	go broker.StartConsumer(rabbitConn, "4", bot, db)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// Цикл обработки сообщений
	for update := range updates {

		// ===== ПЕРЕХВАТ НОВОСТЕЙ ИЗ КАНАЛА =====
		if update.ChannelPost != nil {
			lastChannelID = update.ChannelPost.Chat.ID
			lastMessageID = update.ChannelPost.MessageID
			log.Printf("📥 Получена новая публикация из канала! Message ID: %d", lastMessageID)
			continue
		}

		// ===== ОБРАБОТКА ИНЛАЙН КНОПОК =====
		if update.CallbackQuery != nil {
			userID := update.CallbackQuery.From.ID
			callbackData := update.CallbackQuery.Data

			// 1. Смена языка
			if callbackData == "lang_ru" || callbackData == "lang_uz" {
				langCode := callbackData[5:]
				repository.UpdateLanguage(db, userID, langCode)

				// Проверяем, это регистрация или смена из настроек
				if !repository.IsUserRegistered(db, userID) {
					// ЭТО РЕГИСТРАЦИЯ: переводим на шаг ввода имени
					repository.UpdateUserState(db, userID, "REG_NAME")

					bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "")) // Убираем часики загрузки

					// Запрашиваем имя уже на выбранном языке
					msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, i18n.Get(langCode, "msg_reg_name"))
					msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
					bot.Send(msg)
				} else {
					// ОБЫЧНАЯ СМЕНА В НАСТРОЙКАХ
					responseText := i18n.Get(langCode, "msg_lang_changed")
					bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, responseText))

					msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, responseText)
					msg.ReplyMarkup = handlers.MainMenuKeyboard(langCode)
					bot.Send(msg)
				}
				continue
			}

			// 2. Принятие чата оператором
			if len(callbackData) >= 7 && callbackData[:7] == "accept_" {
				clientIDStr := callbackData[7:]
				clientID, _ := strconv.ParseInt(clientIDStr, 10, 64)
				operatorID := userID

				err := repository.StartChat(db, clientID, operatorID)
				if err == nil {
					bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Чат принят!"))

					// 1. Узнаем язык КЛИЕНТА
					clientLang := repository.GetUserLang(db, clientID)

					// 2. Получаем данные оператора
					opInfo := repository.GetOperatorInfo(db, operatorID)

					// 3. Берем правильный перевод названия отдела из словаря i18n
					deptKey := "dept_" + strconv.Itoa(opInfo.DeptID)
					localizedDeptName := i18n.Get(clientLang, deptKey)

					// 4. Формируем приветствие на языке клиента
					var greeting string
					if clientLang == "uz" {
						greeting = "Assalomu alaykum! Men «" + localizedDeptName + "» bo'limidan " + opInfo.Name + "man.\nSizga yordam bera olishim uchun muammoyingizni tasvirlab bering yoki savolingizni yo'llang."
					} else {
						greeting = "Здравствуйте! Я " + opInfo.Name + " из отдела «" + localizedDeptName + "».\nОпишите Вашу проблему или задайте вопрос, чтобы я смог Вам помочь."
					}

					// 5. Отправляем системное уведомление и приветствие клиенту
					bot.Send(tgbotapi.NewMessage(clientID, i18n.Get(clientLang, "msg_op_connected")))
					bot.Send(tgbotapi.NewMessage(clientID, greeting))

					// 6. Уведомляем оператора и меняем ему клавиатуру
					opMsg := tgbotapi.NewMessage(operatorID, "🟢 Вы успешно приняли чат! Клиенту отправлено авто-приветствие.")
					opMsg.ReplyMarkup = handlers.OperatorInChatKeyboard()
					bot.Send(opMsg)
				} else {
					bot.Send(tgbotapi.NewMessage(operatorID, "❌ Ошибка. Возможно, чат уже принял другой оператор."))
				}
				continue
			}
			continue
		}

		// ===== ОБРАБОТКА ТЕКСТОВЫХ СООБЩЕНИЙ =====
		if update.Message == nil {
			continue
		}

		userID := update.Message.From.ID
		userName := update.Message.From.UserName
		firstName := update.Message.From.FirstName

		// Сохраняем базовую запись пользователя (upsert)
		repository.SaveUser(db, userID, userName, firstName)

		// Получаем актуальный статус из базы данных
		lang := repository.GetUserLang(db, userID)
		state := repository.GetUserState(db, userID)
		isOp := repository.IsOperator(db, userID)

		// ===== ОБРАБОТКА КОМАНДЫ /start =====
		if update.Message.Command() == "start" {
			if isOp {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "👨‍💻 Добро пожаловать в панель оператора! Выберите действие:")
				msg.ReplyMarkup = handlers.OperatorMenuKeyboard(repository.GetOperatorInfo(db, userID).Status)
				bot.Send(msg)
			} else {
				// Если клиент не зарегистрирован — начинаем процесс
				if !repository.IsUserRegistered(db, userID) {
					repository.UpdateUserState(db, userID, "REG_LANG") // НОВОЕ СОСТОЯНИЕ

					// Двуязычное сообщение
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите язык интерфейса / Interfeys tilini tanlang:")
					msg.ReplyMarkup = handlers.SettingsInlineKeyboard() // Выдаем инлайн-кнопки
					bot.Send(msg)
				} else {
					repository.UpdateUserState(db, userID, "MAIN_MENU")
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, i18n.Get(lang, "msg_start_client"))
					msg.ReplyMarkup = handlers.MainMenuKeyboard(lang)
					bot.Send(msg)
				}
			}
			continue
		}

		// ===== LIVE CHAT PROXY (ПЕРЕСЫЛКА СООБЩЕНИЙ) =====
		partnerID := repository.GetPartnerID(db, userID)
		if partnerID != 0 {
			if update.Message.Text == "❌ Завершить текущий чат" || update.Message.Command() == "start" {
				_, err := repository.EndChat(db, userID)
				if err == nil {
					bot.Send(tgbotapi.NewMessage(userID, "🔴 Вы завершили диалог."))
					bot.Send(tgbotapi.NewMessage(partnerID, "🔴 Собеседник завершил диалог. Сессия закрыта."))

					// Обновляем меню собеседнику
					if repository.IsOperator(db, partnerID) {
						msg := tgbotapi.NewMessage(partnerID, "Вы снова ONLINE. Ожидайте новых запросов.")
						msg.ReplyMarkup = handlers.OperatorMenuKeyboard("ONLINE") // Возвращаем полные кнопки
						bot.Send(msg)
					} else {
						pLang := repository.GetUserLang(db, partnerID)
						msg := tgbotapi.NewMessage(partnerID, i18n.Get(pLang, "msg_start_client"))
						msg.ReplyMarkup = handlers.MainMenuKeyboard(pLang)
						bot.Send(msg)
					}

					// Сбрасываем текст команды для инициатора
					if update.Message.Text == "❌ Завершить текущий чат" {
						msg := tgbotapi.NewMessage(userID, "Возврат в меню:")
						if isOp {
							msg.ReplyMarkup = handlers.OperatorMenuKeyboard("ONLINE") // Возвращаем полные кнопки
						} else {
							msg.ReplyMarkup = handlers.MainMenuKeyboard(lang)
						}
						bot.Send(msg)
						continue
					}
				}
			} else {
				// Пересылаем текст собеседнику
				msg := tgbotapi.NewMessage(partnerID, update.Message.Text)
				bot.Send(msg)
				continue
			}
		}

		// ===== ЛОГИКА ДЛЯ ОПЕРАТОРОВ (БЕЗ АКТИВНОГО ЧАТА) =====
		if isOp {
			switch update.Message.Text {

			case "🔴 Стать OFFLINE":
				repository.SetOperatorStatus(db, userID, "OFFLINE")
				msg := tgbotapi.NewMessage(userID, "Ваш статус изменен на OFFLINE. Вы не будете получать новые заявки.")
				msg.ReplyMarkup = handlers.OperatorMenuKeyboard("OFFLINE") // Меняем кнопки
				bot.Send(msg)

			case "🟢 Стать ONLINE":
				repository.SetOperatorStatus(db, userID, "ONLINE")
				msg := tgbotapi.NewMessage(userID, "Ваш статус изменен на ONLINE. Ожидайте новые заявки.")
				msg.ReplyMarkup = handlers.OperatorMenuKeyboard("ONLINE") // Меняем кнопки
				bot.Send(msg)

			case "ℹ️ Мой статус: ONLINE", "ℹ️ Мой статус: OFFLINE":
				info := repository.GetOperatorInfo(db, userID)
				sessionText := "Нет"
				if info.HasSession {
					sessionText = "Да (идет диалог)"
				}

				// Формируем красивую анкету
				text := "👤 Оператор: " + info.Name + "\n" +
					"🏢 Отдел: " + info.DeptName + "\n" +
					"🚦 Статус: " + info.Status + "\n" +
					"💬 Активная сессия: " + sessionText

				bot.Send(tgbotapi.NewMessage(userID, text))

			case "❌ Завершить текущий чат":
				bot.Send(tgbotapi.NewMessage(userID, "У Вас нет активных сессий."))

			default:
				bot.Send(tgbotapi.NewMessage(userID, "Пожалуйста, используйте кнопки меню."))
			}
			continue
		}

		// ===== FSM РЕГИСТРАЦИИ КЛИЕНТА =====
		if state == "REG_LANG" {
			// Если клиент что-то пишет вместо нажатия кнопки языка
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Пожалуйста, выберите язык, нажав на кнопку ниже ⬇️\nIltimos, pastdagi tugmani bosib tilni tanlang ⬇️")
			msg.ReplyMarkup = handlers.SettingsInlineKeyboard()
			bot.Send(msg)
			continue
		}

		if state == "REG_NAME" {
			repository.UpdateUserRegistrationName(db, userID, update.Message.Text)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, i18n.Get(lang, "msg_reg_phone"))
			msg.ReplyMarkup = handlers.RequestPhoneKeyboard(lang)
			bot.Send(msg)
			continue
		}

		if state == "REG_PHONE" {
			phone := ""
			if update.Message.Contact != nil {
				phone = update.Message.Contact.PhoneNumber
			} else {
				phone = update.Message.Text
			}

			repository.UpdateUserRegistrationPhone(db, userID, phone)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, i18n.Get(lang, "msg_start_client"))
			msg.ReplyMarkup = handlers.MainMenuKeyboard(lang)
			bot.Send(msg)
			continue
		}

		// ===== МАРШРУТИЗАЦИЯ ГЛАВНОГО МЕНЮ КЛИЕНТА =====
		text := update.Message.Text

		switch {
		case text == i18n.Get(lang, "btn_contacts"):
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, i18n.Get(lang, "msg_contacts"))
			bot.Send(msg)

		case text == i18n.Get(lang, "btn_settings"):
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, i18n.Get(lang, "msg_select_lang"))
			msg.ReplyMarkup = handlers.SettingsInlineKeyboard()
			bot.Send(msg)

		case text == i18n.Get(lang, "btn_support"):
			repository.UpdateUserState(db, userID, "SELECT_DEPT")
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, i18n.Get(lang, "msg_select_dept"))
			msg.ReplyMarkup = handlers.SupportDepartmentsReplyKeyboard(lang)
			bot.Send(msg)

		// Кнопки выбора отделов
		case text == i18n.Get(lang, "dept_1") || text == i18n.Get(lang, "dept_2") || text == i18n.Get(lang, "dept_3") || text == i18n.Get(lang, "dept_4"):
			deptID := "4"
			switch text {
			case i18n.Get(lang, "dept_1"):
				deptID = "1"
			case i18n.Get(lang, "dept_2"):
				deptID = "2"
			case i18n.Get(lang, "dept_3"):
				deptID = "3"
			}

			repository.UpdateUserState(db, userID, "WAITING")

			waitMsg := "Запрос принят. Ожидайте, ищем свободного оператора..."
			if lang == "uz" {
				waitMsg = "So'rov qabul qilindi. Kutib turing, bo'sh operatorni qidirmoqdamiz..."
			}
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, waitMsg)
			bot.Send(msg)

			broker.PublishTicket(rabbitConn, userID, deptID)

		case text == i18n.Get(lang, "btn_back"):
			repository.UpdateUserState(db, userID, "MAIN_MENU")
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, i18n.Get(lang, "msg_start_client"))
			msg.ReplyMarkup = handlers.MainMenuKeyboard(lang)
			bot.Send(msg)

		case text == i18n.Get(lang, "btn_news"):
			if lastMessageID != 0 {
				forward := tgbotapi.NewForward(update.Message.Chat.ID, lastChannelID, lastMessageID)
				_, err := bot.Send(forward)
				if err != nil {
					log.Println("Ошибка при пересылке новости:", err)
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Не удалось загрузить последнюю новость."))
				}
			} else {
				newsMsg := "Пока новых публикаций нет. Ожидайте обновлений!"
				if lang == "uz" {
					newsMsg = "Hozircha yangi nashrlar yo'q. Yangilanishlarni kuting!"
				}
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, newsMsg))
			}

		default:
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, i18n.Get(lang, "msg_default_text"))
			bot.Send(msg)
		}
	}
}
