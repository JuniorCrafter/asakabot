package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	"asakabot/internal/broker"
	"asakabot/internal/config"
	"asakabot/internal/handlers"
	"asakabot/internal/i18n"
	"asakabot/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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

	bot.Debug = false // Отключаем лишний спам в консоли
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

		// ===== ОБРАБОТКА ИНЛАЙН КНОПОК =====
		if update.CallbackQuery != nil {
			userID := update.CallbackQuery.From.ID
			callbackData := update.CallbackQuery.Data

			// 1. Смена языка
			if callbackData == "lang_ru" || callbackData == "lang_uz" {
				langCode := callbackData[5:]
				repository.UpdateLanguage(db, userID, langCode)

				if !repository.IsUserRegistered(db, userID) {
					repository.UpdateUserState(db, userID, "REG_NAME")
					bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))

					msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, i18n.Get(langCode, "msg_reg_name"))
					msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
					bot.Send(msg)
				} else {
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

					clientLang := repository.GetUserLang(db, clientID)
					opInfo := repository.GetOperatorInfo(db, operatorID)

					deptKey := "dept_" + strconv.Itoa(opInfo.DeptID)
					localizedDeptName := i18n.Get(clientLang, deptKey)

					var greeting string
					if clientLang == "uz" {
						greeting = "Assalomu alaykum! Men «" + localizedDeptName + "» bo'limidan " + opInfo.Name + "man.\nSizga yordam bera olishim uchun muammoyingizni tasvirlab bering yoki savolingizni yo'llang."
					} else {
						greeting = "Здравствуйте! Я " + opInfo.Name + " из отдела «" + localizedDeptName + "».\nОпишите Вашу проблему или задайте вопрос, чтобы я смог Вам помочь."
					}

					bot.Send(tgbotapi.NewMessage(clientID, i18n.Get(clientLang, "msg_op_connected")))
					bot.Send(tgbotapi.NewMessage(clientID, greeting))

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

		// ===== ОБРАБОТКА ТЕКСТОВЫХ И МЕДИА СООБЩЕНИЙ =====
		if update.Message == nil {
			continue
		}

		userID := update.Message.From.ID
		userName := update.Message.From.UserName
		firstName := update.Message.From.FirstName

		repository.SaveUser(db, userID, userName, firstName)

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
				if !repository.IsUserRegistered(db, userID) {
					repository.UpdateUserState(db, userID, "REG_LANG")
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Выберите язык интерфейса / Interfeys tilini tanlang:")
					msg.ReplyMarkup = handlers.SettingsInlineKeyboard()
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

		// ===== ОПРЕДЕЛЕНИЕ ПРАВ АДМИНИСТРАТОРА =====
		adminIDStr := os.Getenv("ADMIN_ID")
		adminID, _ := strconv.ParseInt(adminIDStr, 10, 64)
		isAdmin := (userID == adminID)

		// Вход в панель по команде /admin
		if update.Message.Command() == "admin" && isAdmin {
			repository.UpdateUserState(db, userID, "ADMIN_MAIN")
			msg := tgbotapi.NewMessage(userID, "👑 Добро пожаловать в панель администратора.")
			msg.ReplyMarkup = handlers.AdminMenuKeyboard()
			bot.Send(msg)
			continue
		}

		// ===== ЛОГИКА АДМИНИСТРАТОРА =====
		if isAdmin && len(state) >= 5 && state[:5] == "ADMIN" {
			switch update.Message.Text {

			case "📊 Статистика":
				stats := repository.GetSystemStats(db)
				msg := tgbotapi.NewMessage(userID, stats)
				msg.ParseMode = "Markdown"
				bot.Send(msg)

			case "🔙 Выйти из панели":
				repository.UpdateUserState(db, userID, "MAIN_MENU")
				msg := tgbotapi.NewMessage(userID, "Вы вышли из панели управления.")
				if isOp {
					msg.ReplyMarkup = handlers.OperatorMenuKeyboard(repository.GetOperatorInfo(db, userID).Status)
				} else {
					msg.ReplyMarkup = handlers.MainMenuKeyboard(lang)
				}
				bot.Send(msg)

			case "➕ Добавить оператора":
				repository.UpdateUserState(db, userID, "ADMIN_ADD_OP")
				msg := tgbotapi.NewMessage(userID, "Отправьте данные нового оператора в формате:\n`ID ОТДЕЛ ИМЯ`\n\nПример: `123456789 1 Иван Иванов`")
				msg.ParseMode = "Markdown" // Указываем Markdown правильным способом
				bot.Send(msg)

			case "➖ Удалить оператора":
				repository.UpdateUserState(db, userID, "ADMIN_DEL_OP")
				bot.Send(tgbotapi.NewMessage(userID, "Введите Telegram ID оператора для удаления:"))

			case "🚦 Управление статусами":
				repository.UpdateUserState(db, userID, "ADMIN_FORCE_STATUS")
				msg := tgbotapi.NewMessage(userID, "Введите команду в формате:\n`ID ONLINE` или `ID OFFLINE`\n\nПример: `123456789 OFFLINE`")
				msg.ParseMode = "Markdown" // И здесь тоже
				bot.Send(msg)

			case "🔄 Изменить отдел":
				repository.UpdateUserState(db, userID, "ADMIN_UPDATE_DEPT")
				msg := tgbotapi.NewMessage(userID, "Введите данные в формате:\n`ID_ОПЕРАТОРА НОВЫЙ_ОТДЕЛ`\n\nПример: `123456789 2`")
				msg.ParseMode = "Markdown"
				bot.Send(msg)

			default:
				// Обработка ввода данных администратором
				if state == "ADMIN_ADD_OP" {
					parts := strings.SplitN(update.Message.Text, " ", 3)
					if len(parts) == 3 {
						targetID, _ := strconv.ParseInt(parts[0], 10, 64)
						deptID, _ := strconv.Atoi(parts[1])
						name := parts[2]

						err := repository.AddOperator(db, targetID, name, deptID)
						if err == nil {
							bot.Send(tgbotapi.NewMessage(userID, "✅ Оператор "+name+" успешно добавлен!"))
						} else {
							bot.Send(tgbotapi.NewMessage(userID, "❌ Ошибка базы данных. Возможно, этот ID уже зарегистрирован."))
						}
					} else {
						bot.Send(tgbotapi.NewMessage(userID, "❌ Неверный формат. Ожидалось 3 параметра."))
					}
					repository.UpdateUserState(db, userID, "ADMIN_MAIN")

				} else if state == "ADMIN_DEL_OP" {
					targetID, err := strconv.ParseInt(update.Message.Text, 10, 64)
					if err == nil {
						repository.DeleteOperator(db, targetID)
						bot.Send(tgbotapi.NewMessage(userID, "✅ Оператор удален."))
					} else {
						bot.Send(tgbotapi.NewMessage(userID, "❌ Неверный формат ID."))
					}
					repository.UpdateUserState(db, userID, "ADMIN_MAIN")

				} else if state == "ADMIN_FORCE_STATUS" {
					parts := strings.Split(update.Message.Text, " ")
					if len(parts) == 2 {
						targetID, _ := strconv.ParseInt(parts[0], 10, 64)
						newStatus := strings.ToUpper(parts[1])

						if newStatus == "ONLINE" || newStatus == "OFFLINE" {
							repository.ForceOperatorStatus(db, targetID, newStatus)
							bot.Send(tgbotapi.NewMessage(userID, "✅ Статус оператора изменен на "+newStatus))
						} else {
							bot.Send(tgbotapi.NewMessage(userID, "❌ Статус должен быть ONLINE или OFFLINE."))
						}
					}
				} else if state == "ADMIN_UPDATE_DEPT" {
					parts := strings.Split(update.Message.Text, " ")
					if len(parts) == 2 {
						targetID, _ := strconv.ParseInt(parts[0], 10, 64)
						newDept, _ := strconv.Atoi(parts[1])

						err := repository.UpdateOperatorDept(db, targetID, newDept)
						if err == nil {
							bot.Send(tgbotapi.NewMessage(userID, "✅ Отдел оператора успешно изменен!"))
						} else {
							bot.Send(tgbotapi.NewMessage(userID, "❌ Ошибка базы данных. Проверьте ID оператора."))
						}
					} else {
						bot.Send(tgbotapi.NewMessage(userID, "❌ Неверный формат. Ожидалось 2 параметра."))
					}
					repository.UpdateUserState(db, userID, "ADMIN_MAIN")

				} else {
					msg := tgbotapi.NewMessage(userID, "Используйте кнопки меню администратора.")
					msg.ReplyMarkup = handlers.AdminMenuKeyboard()
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

					if repository.IsOperator(db, partnerID) {
						msg := tgbotapi.NewMessage(partnerID, "Вы снова ONLINE. Ожидайте новых запросов.")
						msg.ReplyMarkup = handlers.OperatorMenuKeyboard("ONLINE")
						bot.Send(msg)
					} else {
						pLang := repository.GetUserLang(db, partnerID)
						msg := tgbotapi.NewMessage(partnerID, i18n.Get(pLang, "msg_start_client"))
						msg.ReplyMarkup = handlers.MainMenuKeyboard(pLang)
						bot.Send(msg)
					}

					if update.Message.Text == "❌ Завершить текущий чат" {
						msg := tgbotapi.NewMessage(userID, "Возврат в меню:")
						if isOp {
							msg.ReplyMarkup = handlers.OperatorMenuKeyboard("ONLINE")
						} else {
							msg.ReplyMarkup = handlers.MainMenuKeyboard(lang)
						}
						bot.Send(msg)
						continue
					}
				}
			} else {
				// 1. Проверяем, какой тип сообщения прислал пользователь
				isAllowed := false

				if update.Message.Photo != nil ||
					update.Message.Video != nil ||
					update.Message.Voice != nil ||
					update.Message.VideoNote != nil ||
					update.Message.Text != "" { // Разрешаем текст
					isAllowed = true
				}

				// 2. Если тип разрешен — копируем
				// 2. Если тип разрешен — копируем и сохраняем
				if isAllowed {
					// --- НАЧАЛО БЛОКА СОХРАНЕНИЯ В ОДНУ СТРОКУ ---
					// Определяем, кто пишет
					senderRole := "Клиент"
					if isOp {
						senderRole = "Оператор"
					}

					// Извлекаем текст (или подпись к фото)
					textToSave := update.Message.Text
					if update.Message.Photo != nil {
						textToSave = "[Фотография] " + update.Message.Caption
					} else if update.Message.Video != nil {
						textToSave = "[Видео] " + update.Message.Caption
					} else if update.Message.Voice != nil {
						textToSave = "[Голосовое сообщение]"
					} else if update.Message.VideoNote != nil {
						textToSave = "[Видеосообщение (кружок)]"
					}

					// Если сообщение не полностью пустое, записываем его в базу
					if textToSave != "" && textToSave != "[Фотография] " && textToSave != "[Видео] " {
						repository.AppendToChatLog(db, userID, senderRole, textToSave)
					}
					// --- КОНЕЦ БЛОКА СОХРАНЕНИЯ ---

					// Отправляем само сообщение собеседнику
					copyMsg := tgbotapi.NewCopyMessage(partnerID, update.Message.Chat.ID, update.Message.MessageID)
					_, err := bot.Send(copyMsg)
					if err != nil {
						log.Printf("Ошибка пересылки медиа: %v", err)
					}
				} else {
					// 3. Если прислали документ, стикер или другой файл — уведомляем
					bot.Send(tgbotapi.NewMessage(userID, "❌ Извините, этот тип файлов запрещен. Вы можете отправлять только фото, видео, голосовые и текст."))
				}
				continue
			}
		}

		// ===== ЛОГИКА ДЛЯ ОПЕРАТОРОВ (БЕЗ АКТИВНОГО ЧАТА) =====
		if isOp {
			switch update.Message.Text {
			case "🔴 Стать OFFLINE":
				repository.SetOperatorStatus(db, userID, "OFFLINE")
				msg := tgbotapi.NewMessage(userID, "Ваш статус изменен на OFFLINE.")
				msg.ReplyMarkup = handlers.OperatorMenuKeyboard("OFFLINE")
				bot.Send(msg)

			case "🟢 Стать ONLINE":
				repository.SetOperatorStatus(db, userID, "ONLINE")
				msg := tgbotapi.NewMessage(userID, "Ваш статус изменен на ONLINE.")
				msg.ReplyMarkup = handlers.OperatorMenuKeyboard("ONLINE")
				bot.Send(msg)

			case "ℹ️ Мой статус: ONLINE", "ℹ️ Мой статус: OFFLINE":
				info := repository.GetOperatorInfo(db, userID)
				sessionText := "Нет"
				if info.HasSession {
					sessionText = "Да (идет диалог)"
				}
				text := "👤 Оператор: " + info.Name + "\n🏢 Отдел: " + info.DeptName + "\n🚦 Статус: " + info.Status + "\n💬 Сессия: " + sessionText
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
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Пожалуйста, выберите язык ⬇️\nIltimos, tilni tanlang ⬇️")
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
			repository.UpdateUserState(db, userID, "MAIN_MENU") // Добавлено, чтобы клиент вышел из регистрации
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, i18n.Get(lang, "msg_start_client"))
			msg.ReplyMarkup = handlers.MainMenuKeyboard(lang)
			bot.Send(msg)
			continue
		}

		// ===== МАРШРУТИЗАЦИЯ ГЛАВНОГО МЕНЮ КЛИЕНТА =====
		text := update.Message.Text

		switch {
		// ИСПРАВЛЕНИЕ 2: БЕЗОПАСНЫЙ ВЫВОД КОНТАКТОВ
		case text == i18n.Get(lang, "btn_contacts"):
			contactsText := i18n.Get(lang, "msg_contacts")

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, contactsText)
			msg.ParseMode = "Markdown"
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

		case text == i18n.Get(lang, "btn_about"):
			infoText := "🏦 *Asaka Bank Nurafshon*\n\nМы предоставляем современные банковские услуги. Наш филиал всегда открыт для вас!\n\n📍 Наш адрес: г. Нурафшон, ул. Ташкентская, 1."
			infoMsg := tgbotapi.NewMessage(update.Message.Chat.ID, infoText)
			infoMsg.ParseMode = "Markdown"

			// Инлайн-кнопка со ссылкой на канал
			infoMsg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonURL("📢 Наш Telegram канал", "https://t.me/Asakabank_Nurafshon_BXM"),
				),
			)
			bot.Send(infoMsg)

			// Отправка геолокации
			locMsg := tgbotapi.NewLocation(update.Message.Chat.ID, 41.032971, 69.359179)
			bot.Send(locMsg)

		default:
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, i18n.Get(lang, "msg_default_text"))
			bot.Send(msg)
		}
	}
}
