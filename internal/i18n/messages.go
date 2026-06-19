package i18n

// Messages holds every user-facing string for one UI language. Keeping them in a
// single struct lets additional UI languages be added without touching screens.
type Messages struct {
	AppName  string
	Tagline  string // "🇪🇸 → 🇯🇵"
	YesNoYes string

	// Menu
	MenuPrompt     string
	ItemKana       string
	ItemFlashcards string
	ItemQuiz       string
	ItemStats      string
	ItemQuit       string
	MenuHelp       string
	ComingSoon     string

	// Progress badge
	LevelLabel    string // "Nivel"
	TowardLabel   string // "hacia"
	StreakLabel   string // "Racha"
	DaysSuffix    string // "días"
	LearnedSuffix string // "palabras aprendidas"
}

// ES is the Spanish localization, used by default in v1.
var ES = Messages{
	AppName:  "Polyglot",
	Tagline:  "🇪🇸 → 🇯🇵",
	YesNoYes: "Sí",

	MenuPrompt:     "¿Qué quieres estudiar hoy?",
	ItemKana:       "Entrenador de Kana",
	ItemFlashcards: "Flashcards (repaso espaciado)",
	ItemQuiz:       "Quiz de opción múltiple",
	ItemStats:      "Mis estadísticas",
	ItemQuit:       "Salir",
	MenuHelp:       "↑/↓ moverse · enter elegir · q salir",
	ComingSoon:     "Próximamente ✨",

	LevelLabel:    "Nivel",
	TowardLabel:   "hacia",
	StreakLabel:   "Racha",
	DaysSuffix:    "días",
	LearnedSuffix: "palabras aprendidas",
}

// Default is the active UI language.
var Default = ES
