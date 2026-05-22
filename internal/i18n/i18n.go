package i18n

// T — это глобальная карта переводов
var T = map[string]map[string]string{
	"ru": {
		// Кнопки главного меню
		"btn_contacts": "📞 Контакты",
		"btn_settings": "⚙️ Настройки",
		"btn_support":  "🎧 Поддержка",
		"btn_news":     "📰 Новости",
		"btn_back":     "↩️ Назад",

		// Кнопки отделов поддержки
		"dept_1": "🏦 Махалла банкирлари",
		"dept_2": "⚖️ Юридические услуги",
		"dept_3": "🛍 Розничные услуги",
		"dept_4": "❓ Общие вопросы",

		// Сообщения
		"msg_start_client": "Здравствуйте! Я бот поддержки банка. Выберите нужный раздел:",
		"msg_select_lang":  "Выберите язык интерфейса / Interfeys tilini tanlang:",
		"msg_lang_changed": "Язык успешно изменен на Русский 🇷🇺",
		"msg_contacts":     "Свяжитесь с нами:\n📞 Телефон: +998 71 123-45-67\n📧 Email: info@asakabank.uz",
		"msg_select_dept":  "Выберите отдел для связи со специалистом:",
		"msg_default_text": "Пожалуйста, используйте кнопки меню внизу экрана.",

		// Регистрация
		"msg_reg_name":    "Добро пожаловать! Для продолжения, пожалуйста, введите Ваши реальные Имя и Фамилию:",
		"msg_reg_phone":   "Спасибо! Теперь нажмите на кнопку ниже, чтобы поделиться номером телефона, или отправьте его текстом:",
		"btn_share_phone": "📱 Поделиться контактом",
	},
	"uz": {
		// Кнопки главного меню
		"btn_contacts": "📞 Kontaktlar",
		"btn_settings": "⚙️ Sozlamalar",
		"btn_support":  "🎧 Yordam",
		"btn_news":     "📰 Yangiliklar",
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
		"msg_contacts":     "Biz bilan bog'lanish:\n📞 Telefon: +998 71 123-45-67\n📧 Email: info@asakabank.uz",
		"msg_select_dept":  "Mutaxassis bilan bog'lanish uchun bo'limni tanlang:",
		"msg_default_text": "Iltimos, ekranning pastki qismidagi menyu tugmalaridan foydalaning.",

		// Регистрация
		"msg_reg_name":    "Xush kelibsiz! Davom etish uchun iltimos, haqiqiy Ism va Familiyangizni kiriting:",
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
