package i18n

// Messages holds every user-facing string for one UI language. Keeping them in a
// single struct lets additional UI languages be added without touching screens.
type Messages struct {
	AppName string
	Tagline string // "🇪🇸 → 🇯🇵"

	// Menu
	MenuPrompt     string
	ItemKana       string
	ItemFlashcards string
	ItemQuiz       string
	ItemStats      string
	ItemQuit       string
	MenuHelp       string

	// Progress badge
	LevelLabel    string // "Nivel"
	TowardLabel   string // "hacia"
	StreakLabel   string // "Racha"
	DaysSuffix    string // "días"
	LearnedSuffix string // "palabras aprendidas"

	// Shared study UI
	ChoiceHelp   string
	ContinueHelp string
	RestartHelp  string
	BackHelp     string
	SessionDone  string
	ScoreLabel   string

	// Kana trainer
	KanaTitle  string
	KanaPrompt string

	// Quiz
	QuizTitle       string
	QuizQuestionFmt string // "¿Cómo se dice \"%s\" en japonés?"
	ReviewLabel     string

	// Flashcards
	FlashTitle    string
	RevealHelp    string
	GradePrompt   string
	GradeAgain    string
	GradeHard     string
	GradeGood     string
	GradeEasy     string
	ReviewedLabel string
	NothingDue    string
	Today         string
	DayShort      string

	// Stats
	StatsTitle    string
	BestLabel     string
	HiraganaLabel string
	KatakanaLabel string
}

// ES is the Spanish localization, used by default in v1.
var ES = Messages{
	AppName: "Polyglot",
	Tagline: "🇪🇸 → 🇯🇵",

	MenuPrompt:     "¿Qué quieres estudiar hoy?",
	ItemKana:       "Entrenador de Kana",
	ItemFlashcards: "Flashcards (repaso espaciado)",
	ItemQuiz:       "Quiz de opción múltiple",
	ItemStats:      "Mis estadísticas",
	ItemQuit:       "Salir",
	MenuHelp:       "↑/↓ moverse · enter elegir · q salir",

	LevelLabel:    "Nivel",
	TowardLabel:   "hacia",
	StreakLabel:   "Racha",
	DaysSuffix:    "días",
	LearnedSuffix: "palabras aprendidas",

	ChoiceHelp:   "1-4 elegir · ↑/↓ mover · enter confirmar · esc menú",
	ContinueHelp: "enter continuar · esc menú",
	RestartHelp:  "enter reiniciar · esc menú",
	BackHelp:     "esc volver al menú",
	SessionDone:  "¡Sesión completada! 🎉",
	ScoreLabel:   "Aciertos",

	KanaTitle:  "Entrenador de Kana",
	KanaPrompt: "¿Cómo se lee?",

	QuizTitle:       "Quiz",
	QuizQuestionFmt: "¿Cómo se dice \"%s\" en japonés?",
	ReviewLabel:     "Repasa",

	FlashTitle:    "Flashcards",
	RevealHelp:    "espacio revelar · esc menú",
	GradePrompt:   "¿Qué tal lo recordaste?",
	GradeAgain:    "Otra vez",
	GradeHard:     "Difícil",
	GradeGood:     "Bien",
	GradeEasy:     "Fácil",
	ReviewedLabel: "Tarjetas repasadas",
	NothingDue:    "No hay tarjetas para repasar ahora. ¡Vuelve más tarde! 🌙",
	Today:         "hoy",
	DayShort:      "d",

	StatsTitle:    "Mis estadísticas",
	BestLabel:     "récord",
	HiraganaLabel: "Hiragana",
	KatakanaLabel: "Katakana",
}

// Default is the active UI language.
var Default = ES
