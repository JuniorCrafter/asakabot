package i18n

// T — это глобальная карта переводов
var T = map[string]map[string]string{
	"ru": {
		// Кнопки главного меню
		"btn_contacts": "📞 Контакты",
		"btn_settings": "⚙️ Настройки",
		"btn_support":  "🎧 Поддержка",
		"btn_about":    "🏢 О нас",
		"btn_back":     "↩️ Назад",

		// Кнопки отделов поддержки
		"dept_1": "🏦 Махалла банкирлари",
		"dept_2": "⚖️ Юридические услуги",
		"dept_3": "🛍 Чакана хизмат",
		"dept_4": "❓ Общие вопросы",

		// Сообщения
		"msg_start_client": "Здравствуйте! Я бот поддержки банка. Выберите нужный раздел:",
		"msg_select_lang":  "Выберите язык интерфейса / Interfeys tilini tanlang:",
		"msg_lang_changed": "Язык успешно изменен на Русский 🇷🇺",
		"msg_contacts":     "Свяжитесь с нами:\n\n📞 Телефон: +998781476410\n\n📞 Телефон: +998781476417 \n\n📞 Телефон: +998781476418 \n\n📧 Email: nurafshan@asakabank.uz",
		"msg_select_dept":  "Выберите отдел для связи со оператором:",
		"msg_default_text": "Пожалуйста, используйте кнопки меню внизу экрана.",
		"msg_op_connected": "🟢 Оператор подключился! Можете рассказать о вашей возникшейся проблеме. Чтобы завершить чат, отправьте /start",

		// Регистрация
		"msg_reg_name":    "Добро пожаловать! Для продолжения, пожалуйста, введите Ваши Имя и Фамилию:",
		"msg_reg_phone":   "Спасибо! Теперь нажмите на кнопку ниже, чтобы поделиться номером телефона, или отправьте его текстом:",
		"btn_share_phone": "📱 Поделиться контактом",
	},
	"uz": {
		// Кнопки главного меню
		"btn_contacts": "📞 Kontaktlar",
		"btn_settings": "⚙️ Sozlamalar",
		"btn_support":  "🎧 Yordam",
		"btn_about":    "🏢 Biz haqimizda",
		"btn_back":     "↩️ Orqaga",

		// Кнопки отделов поддержки
		"dept_1": "🏦 Mahalla bankirlari",
		"dept_2": "⚖️ Yuridik xizmatlar",
		"dept_3": "🛍 Chakana xizmat",
		"dept_4": "❓ Umumiy savollar",

		// Сообщения
		"msg_start_client": "Assalomu alaykum! Men bank qo'llab-quvvatlash botiman. Kerakli bo'limni tanlang:",
		"msg_select_lang":  "Выберите язык интерфейса / Interfeys tilini tanlang:",
		"msg_lang_changed": "Til muvaffaqiyatli O'zbek tiliga o'zgartirildi 🇺🇿",
		"msg_contacts":     "Biz bilan bog'lanish:\n📞 Telefon: +998781476410\n📞 Telefon: +998781476417\n📞 Telefon: +998781476418\n📧 Email: nurafshan@asakabank.uz",
		"msg_select_dept":  "Operator bilan bog'lanish uchun bo'limni tanlang:",
		"msg_default_text": "Iltimos, ekranning pastki qismidagi menyu tugmalaridan foydalaning.",
		"msg_op_connected": "🟢 Operator ulandi! Sizda yuz bergan muammo haqida aytib berishingiz mumkin. Chatni yakunlash uchun /start ni yuboring",

		// Регистрация
		"msg_reg_name":    "Xush kelibsiz! Davom etish uchun iltimos, Ism va Familiyangizni kiriting:",
		"msg_reg_phone":   "Rahmat! Endi telefon raqamingizni yuborish uchun pastdagi tugmani bosing yoki raqamingizni yozib yuboring:",
		"btn_share_phone": "📱 Kontaktni ulashish",
	},
}

// Get возвращает строку на нужном языке по ключу
func Get(lang, key string) string {
	if lang != "ru" && lang != "uz" {
		lang = "ru" // Язык по умолчанию
	}
	if text, ok := T[lang][key]; ok {
		return text
	}
	return key
}
