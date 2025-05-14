package i18n

// LoadDefaultTranslations loads the default translations for the bot
func LoadDefaultTranslations(translator *Translator) {
	// English translations (default)
	translator.LoadTranslations("en", map[string]string{
		"welcome":                "Welcome to the Cocktail Bot! Send your email to check if you're eligible for a free cocktail.",
		"invalid_email":          "That doesn't look like a valid email address. Please send a properly formatted email (e.g., example@domain.com).",
		"unknown_command":        "Unknown command. Please send your email to check eligibility or use /help for more information.",
		"rate_limited":           "You've made too many requests. Please try again in a few minutes.",
		"email_not_found":        "Email is not in database.",
		"system_unavailable":     "Sorry, our system is temporarily unavailable. Please try again later.",
		"already_redeemed":       "Email found, but free cocktail already consumed on {date}.",
		"eligible":               "Email found! You're eligible for a free cocktail.",
		"error_occurred":         "Sorry, an error occurred. Please try again later.",
		"email_not_cached":       "Sorry, I can't find your email. Please try again.",
		"redemption_success":     "Enjoy your free cocktail! Redeemed on {date}.",
		"skip_redemption":        "You've chosen to skip the cocktail redemption. You can check again later.",
		"button_redeem":          "Get Cocktail",
		"button_skip":            "Skip",
		"help_message":           "Here's how to use the Cocktail Bot:\n\n• Send your email address to check if you're eligible for a free cocktail\n• If eligible, you'll receive options to redeem or skip\n• Choose \"Get Cocktail\" to redeem your free drink\n• Each email can only be redeemed once\n\nCommands:\n/start - Start the bot\n/help - Show this help message\n/language - Change language\n\nSend an email address to begin!",
		"language_command":       "Please select your preferred language:",
		"language_set":           "Language set to English.",
		"language_not_supported": "Sorry, this language is not supported yet.",
	})

	// Spanish translations
	translator.LoadTranslations("es", map[string]string{
		"welcome":                "¡Bienvenido al Bot de Cócteles! Envía tu correo electrónico para verificar si eres elegible para un cóctel gratis.",
		"invalid_email":          "Eso no parece una dirección de correo electrónico válida. Por favor, envía un correo con formato correcto (ej: ejemplo@dominio.com).",
		"unknown_command":        "Comando desconocido. Por favor, envía tu correo electrónico para verificar elegibilidad o usa /help para más información.",
		"rate_limited":           "Has hecho demasiadas solicitudes. Por favor, inténtalo de nuevo en unos minutos.",
		"email_not_found":        "El correo no está en la base de datos.",
		"system_unavailable":     "Lo sentimos, nuestro sistema está temporalmente no disponible. Por favor, inténtalo más tarde.",
		"already_redeemed":       "Correo encontrado, pero el cóctel gratis ya fue consumido el {date}.",
		"eligible":               "¡Correo encontrado! Eres elegible para un cóctel gratis.",
		"error_occurred":         "Lo sentimos, ocurrió un error. Por favor, inténtalo de nuevo más tarde.",
		"email_not_cached":       "Lo siento, no puedo encontrar tu correo. Por favor, inténtalo de nuevo.",
		"redemption_success":     "¡Disfruta tu cóctel gratis! Canjeado el {date}.",
		"skip_redemption":        "Has elegido saltar el canje del cóctel. Puedes verificar nuevamente más tarde.",
		"button_redeem":          "Obtener Cóctel",
		"button_skip":            "Saltar",
		"help_message":           "Aquí tienes cómo usar el Bot de Cócteles:\n\n• Envía tu dirección de correo para verificar si eres elegible para un cóctel gratis\n• Si eres elegible, recibirás opciones para canjear o saltar\n• Elige \"Obtener Cóctel\" para canjear tu bebida gratis\n• Cada correo solo puede ser canjeado una vez\n\nComandos:\n/start - Iniciar el bot\n/help - Mostrar este mensaje de ayuda\n/language - Cambiar idioma\n\n¡Envía una dirección de correo para comenzar!",
		"language_command":       "Por favor, selecciona tu idioma preferido:",
		"language_set":           "Idioma establecido a Español.",
		"language_not_supported": "Lo sentimos, este idioma aún no está soportado.",
	})

	// French translations
	translator.LoadTranslations("fr", map[string]string{
		"welcome":                "Bienvenue sur le Bot Cocktail ! Envoyez votre email pour vérifier si vous êtes éligible pour un cocktail gratuit.",
		"invalid_email":          "Cela ne ressemble pas à une adresse email valide. Veuillez envoyer un email correctement formaté (ex : exemple@domaine.com).",
		"unknown_command":        "Commande inconnue. Veuillez envoyer votre email pour vérifier l'éligibilité ou utiliser /help pour plus d'informations.",
		"rate_limited":           "Vous avez fait trop de demandes. Veuillez réessayer dans quelques minutes.",
		"email_not_found":        "Email non trouvé dans la base de données.",
		"system_unavailable":     "Désolé, notre système est temporairement indisponible. Veuillez réessayer plus tard.",
		"already_redeemed":       "Email trouvé, mais le cocktail gratuit a déjà été consommé le {date}.",
		"eligible":               "Email trouvé ! Vous êtes éligible pour un cocktail gratuit.",
		"error_occurred":         "Désolé, une erreur s'est produite. Veuillez réessayer plus tard.",
		"email_not_cached":       "Désolé, je ne trouve pas votre email. Veuillez réessayer.",
		"redemption_success":     "Profitez de votre cocktail gratuit ! Échangé le {date}.",
		"skip_redemption":        "Vous avez choisi de sauter l'échange de cocktail. Vous pouvez vérifier à nouveau plus tard.",
		"button_redeem":          "Obtenir Cocktail",
		"button_skip":            "Sauter",
		"help_message":           "Voici comment utiliser le Bot Cocktail :\n\n• Envoyez votre adresse email pour vérifier si vous êtes éligible pour un cocktail gratuit\n• Si éligible, vous recevrez des options pour échanger ou sauter\n• Choisissez \"Obtenir Cocktail\" pour échanger votre boisson gratuite\n• Chaque email ne peut être échangé qu'une seule fois\n\nCommandes :\n/start - Démarrer le bot\n/help - Afficher ce message d'aide\n/language - Changer de langue\n\nEnvoyez une adresse email pour commencer !",
		"language_command":       "Veuillez sélectionner votre langue préférée :",
		"language_set":           "Langue définie sur Français.",
		"language_not_supported": "Désolé, cette langue n'est pas encore prise en charge.",
	})

	// German translations
	translator.LoadTranslations("de", map[string]string{
		"welcome":                "Willkommen beim Cocktail-Bot! Senden Sie Ihre E-Mail, um zu prüfen, ob Sie für einen kostenlosen Cocktail in Frage kommen.",
		"invalid_email":          "Das sieht nicht nach einer gültigen E-Mail-Adresse aus. Bitte senden Sie eine korrekt formatierte E-Mail (z.B. beispiel@domain.com).",
		"unknown_command":        "Unbekannter Befehl. Bitte senden Sie Ihre E-Mail, um die Berechtigung zu prüfen, oder verwenden Sie /help für weitere Informationen.",
		"rate_limited":           "Sie haben zu viele Anfragen gestellt. Bitte versuchen Sie es in einigen Minuten erneut.",
		"email_not_found":        "E-Mail nicht in der Datenbank gefunden.",
		"system_unavailable":     "Entschuldigung, unser System ist vorübergehend nicht verfügbar. Bitte versuchen Sie es später erneut.",
		"already_redeemed":       "E-Mail gefunden, aber der kostenlose Cocktail wurde bereits am {date} konsumiert.",
		"eligible":               "E-Mail gefunden! Sie haben Anspruch auf einen kostenlosen Cocktail.",
		"error_occurred":         "Entschuldigung, ein Fehler ist aufgetreten. Bitte versuchen Sie es später erneut.",
		"email_not_cached":       "Entschuldigung, ich kann Ihre E-Mail nicht finden. Bitte versuchen Sie es erneut.",
		"redemption_success":     "Genießen Sie Ihren kostenlosen Cocktail! Eingelöst am {date}.",
		"skip_redemption":        "Sie haben sich entschieden, die Cocktail-Einlösung zu überspringen. Sie können später erneut prüfen.",
		"button_redeem":          "Cocktail erhalten",
		"button_skip":            "Überspringen",
		"help_message":           "Hier ist, wie Sie den Cocktail-Bot verwenden können:\n\n• Senden Sie Ihre E-Mail-Adresse, um zu prüfen, ob Sie für einen kostenlosen Cocktail berechtigt sind\n• Wenn berechtigt, erhalten Sie Optionen zum Einlösen oder Überspringen\n• Wählen Sie \"Cocktail erhalten\", um Ihr kostenloses Getränk einzulösen\n• Jede E-Mail kann nur einmal eingelöst werden\n\nBefehle:\n/start - Bot starten\n/help - Diese Hilfemeldung anzeigen\n/language - Sprache ändern\n\nSenden Sie eine E-Mail-Adresse, um zu beginnen!",
		"language_command":       "Bitte wählen Sie Ihre bevorzugte Sprache:",
		"language_set":           "Sprache auf Deutsch eingestellt.",
		"language_not_supported": "Entschuldigung, diese Sprache wird noch nicht unterstützt.",
	})

	// Russian translations
	translator.LoadTranslations("ru", map[string]string{
		"welcome":                "Добро пожаловать в Cocktail Bot! Отправьте свою электронную почту, чтобы проверить, имеете ли вы право на бесплатный коктейль.",
		"invalid_email":          "Это не похоже на действительный адрес электронной почты. Пожалуйста, отправьте правильно отформатированный email (например, example@domain.com).",
		"unknown_command":        "Неизвестная команда. Пожалуйста, отправьте свой email для проверки права или используйте /help для получения дополнительной информации.",
		"rate_limited":           "Вы сделали слишком много запросов. Пожалуйста, повторите попытку через несколько минут.",
		"email_not_found":        "Email не найден в базе данных.",
		"system_unavailable":     "Извините, наша система временно недоступна. Пожалуйста, повторите попытку позже.",
		"already_redeemed":       "Email найден, но бесплатный коктейль уже был использован {date}.",
		"eligible":               "Email найден! Вы имеете право на бесплатный коктейль.",
		"error_occurred":         "Извините, произошла ошибка. Пожалуйста, повторите попытку позже.",
		"email_not_cached":       "Извините, я не могу найти ваш email. Пожалуйста, повторите попытку.",
		"redemption_success":     "Наслаждайтесь вашим бесплатным коктейлем! Получено {date}.",
		"skip_redemption":        "Вы решили пропустить получение коктейля. Вы можете проверить снова позже.",
		"button_redeem":          "Получить коктейль",
		"button_skip":            "Пропустить",
		"help_message":           "Вот как использовать Cocktail Bot:\n\n• Отправьте свой адрес электронной почты, чтобы проверить, имеете ли вы право на бесплатный коктейль\n• Если вы имеете право, вы получите варианты использования или пропуска\n• Выберите \"Получить коктейль\", чтобы получить бесплатный напиток\n• Каждый email может быть использован только один раз\n\nКоманды:\n/start - Запустить бота\n/help - Показать это сообщение справки\n/language - Изменить язык\n\nОтправьте адрес электронной почты, чтобы начать!",
		"language_command":       "Пожалуйста, выберите предпочитаемый язык:",
		"language_set":           "Язык установлен на Русский.",
		"language_not_supported": "Извините, этот язык еще не поддерживается.",
	})

	// Serbian translations
	translator.LoadTranslations("sr", map[string]string{
		"welcome":                "Dobrodošli u Cocktail Bot! Pošaljite svoju e-mail adresu da proverite da li imate pravo na besplatni koktel.",
		"invalid_email":          "Ovo ne izgleda kao validna e-mail adresa. Molimo vas pošaljite pravilno formatiranu e-mail adresu (npr. primer@domen.com).",
		"unknown_command":        "Nepoznata komanda. Molimo vas pošaljite svoju e-mail adresu da proverite podobnost ili koristite /help za više informacija.",
		"rate_limited":           "Napravili ste previše zahteva. Molimo vas pokušajte ponovo za nekoliko minuta.",
		"email_not_found":        "E-mail nije pronađen u bazi podataka.",
		"system_unavailable":     "Žao nam je, naš sistem je trenutno nedostupan. Molimo vas pokušajte ponovo kasnije.",
		"already_redeemed":       "E-mail pronađen, ali besplatni koktel je već iskorišćen {date}.",
		"eligible":               "E-mail pronađen! Imate pravo na besplatni koktel.",
		"error_occurred":         "Žao nam je, došlo je do greške. Molimo vas pokušajte ponovo kasnije.",
		"email_not_cached":       "Žao mi je, ne mogu da pronađem vašu e-mail adresu. Molimo vas pokušajte ponovo.",
		"redemption_success":     "Uživajte u vašem besplatnom koktelu! Iskorišćeno {date}.",
		"skip_redemption":        "Izabrali ste da preskočite iskorišćavanje koktela. Možete proveriti ponovo kasnije.",
		"button_redeem":          "Uzmi Koktel",
		"button_skip":            "Preskoči",
		"help_message":           "Evo kako koristiti Cocktail Bot:\n\n• Pošaljite svoju e-mail adresu da proverite da li imate pravo na besplatni koktel\n• Ako imate pravo, dobićete opcije za iskorišćavanje ili preskakanje\n• Izaberite \"Uzmi Koktel\" da iskoristite svoje besplatno piće\n• Svaka e-mail adresa može biti iskorišćena samo jednom\n\nKomande:\n/start - Pokrenite bota\n/help - Prikažite ovu poruku za pomoć\n/language - Promenite jezik\n\nPošaljite e-mail adresu da počnete!",
		"language_command":       "Molimo izaberite vaš željeni jezik:",
		"language_set":           "Jezik podešen na Srpski.",
		"language_not_supported": "Žao nam je, ovaj jezik još uvek nije podržan.",
	})
}
