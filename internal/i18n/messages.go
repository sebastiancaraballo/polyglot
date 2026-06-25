package i18n

// Messages holds every user-facing string for one UI language. Keeping them in a
// single struct lets additional UI languages be added without touching screens.
type Messages struct {
	AppName string
	Tagline string // "es → ja"

	// Menu
	ItemKana       string
	ItemKanaChart  string
	ItemFlashcards string
	ItemReview     string
	ItemQuiz       string
	ItemStats      string
	ItemSettings   string
	ItemQuit       string
	SwitchProfile  string
	MenuHelp       string
	LockedLabel    string // shown beside a menu item gated behind kana fluency
	ReadingLocked  string // hint when a learner opens a locked reading activity

	// Settings
	SettingsTitle        string
	SettingsHelp         string
	ShowRomajiLabel      string
	OptionOn             string
	OptionOff            string
	DeleteProfile        string
	DeleteProfileWarning string
	ConfirmDeleteProfile string
	DeleteAllData        string
	DeleteAllWarning     string
	ConfirmDelete        string
	CancelLabel          string
	ConfirmHelp          string

	// Profiles
	ProfileNameTitle       string
	ProfileNamePrompt      string
	ProfileNamePlaceholder string
	ProfileNameEmpty       string
	ProfileNameTooLongFmt  string
	ProfileNameInvalid     string
	ProfileNameHelpFirst   string
	ProfileNameHelpCancel  string
	ProfileCreateError     string
	ProfilesTitle          string
	ProfileCreateNew       string
	ActiveProfileLabel     string
	ProfilesHelp           string
	NoProfiles             string

	// Progress badge
	XPLabel       string // "XP"
	StreakLabel   string // "Racha"
	DaysSuffix    string // "días"
	LearnedSuffix string // "tarjetas aprendidas"

	// Shared study UI
	ChoiceHelp   string
	ContinueHelp string
	RestartHelp  string
	BackHelp     string
	SessionDone  string
	ScoreLabel   string

	// Kana trainer
	KanaTitle       string
	KanaPrompt      string
	KanaGroupAll    string
	KanaPickHelp    string
	KanaFluent      string // badge on a fully-mastered group
	KanaMasteredFmt string // "%d/%d" mastered count
	KanaLockedHint  string // why a katakana group is locked
	FluentBadge     string // syllabary-fluency badge on the summary screen

	// Kana chart
	KanaChartTitle string
	KanaChartHelp  string
	KanaBasic      string
	KanaVoiced     string
	KanaCombo      string

	// Quiz
	QuizTitle       string
	QuizQuestionFmt string // "¿Cómo se dice \"%s\" en japonés?"
	ReviewLabel     string

	// Flashcards / Review
	FlashTitle        string
	ReviewScreenTitle string
	RevealHelp        string
	GradePrompt       string
	GradeAgain        string
	GradeHard         string
	GradeGood         string
	GradeEasy         string
	ReviewedLabel     string
	NothingDue        string
	Today             string
	DayShort          string

	// Stats
	StatsTitle    string
	BestLabel     string
	HiraganaLabel string
	KatakanaLabel string

	// Onboarding
	WelcomeTitle    string
	WelcomeIntro    string
	ControlsTitle   string
	ControlsKeys    []string
	WelcomeNext     string
	PracticeTitle   string
	SampleWord      string
	SampleRomaji    string
	SamplePrompt    string
	SampleOptions   []string
	SampleCorrect   int
	SampleHint      string
	PracticeCorrect string
	PracticeRetry   string
	PracticeNext    string
	DoneTitle       string
	DoneRecommend   string
	DoneNext        string
}

// ES is the Spanish localization, used by default in v1.
var ES = Messages{
	AppName: "Polyglot",
	Tagline: "es → ja",

	ItemKana:       "Entrenador de Kana",
	ItemKanaChart:  "Tabla de Kana",
	ItemFlashcards: "Flashcards",
	ItemReview:     "Repaso",
	ItemQuiz:       "Quiz de opción múltiple",
	ItemStats:      "Mis estadísticas",
	ItemSettings:   "Ajustes",
	ItemQuit:       "Salir",
	SwitchProfile:  "Cambiar de perfil",
	MenuHelp:       "↑/↓ moverse · ENTER elegir/cambiar perfil · Q salir",
	LockedLabel:    "bloqueado",
	ReadingLocked:  "Aprende a leer los kana con fluidez primero.",

	SettingsTitle:        "Ajustes",
	SettingsHelp:         "↑/↓ moverse · ENTER cambiar/confirmar · ESC volver",
	ShowRomajiLabel:      "Mostrar romaji",
	OptionOn:             "Sí",
	OptionOff:            "No",
	DeleteProfile:        "Borrar mi perfil",
	DeleteProfileWarning: "Esto borra solo el perfil actual y su progreso. No se puede deshacer.",
	ConfirmDeleteProfile: "Sí, borrar mi perfil",
	DeleteAllData:        "Borrar todos los datos",
	DeleteAllWarning:     "Esto borra todos los perfiles, progreso y estadísticas. No se puede deshacer.",
	ConfirmDelete:        "Sí, borrar todo",
	CancelLabel:          "Cancelar",
	ConfirmHelp:          "↑/↓ elegir · ENTER confirmar · ESC cancelar",

	ProfileNameTitle:       "Crea tu perfil",
	ProfileNamePrompt:      "¿Cómo te llamas?",
	ProfileNamePlaceholder: "Tu nombre",
	ProfileNameEmpty:       "Escribe un nombre.",
	ProfileNameTooLongFmt:  "Máximo %d caracteres.",
	ProfileNameInvalid:     "Usa letras, espacios o puntuación de nombre.",
	ProfileNameHelpFirst:   "Escribe tu nombre · ENTER crear perfil",
	ProfileNameHelpCancel:  "Escribe tu nombre · ENTER crear perfil · ESC cancelar",
	ProfileCreateError:     "No pude crear el perfil.",
	ProfilesTitle:          "Perfiles",
	ProfileCreateNew:       "＋ Crear nuevo perfil",
	ActiveProfileLabel:     "actual",
	ProfilesHelp:           "↑/↓ mover · ENTER elegir · ESC menú",
	NoProfiles:             "No hay perfiles todavía.",

	XPLabel:       "XP",
	StreakLabel:   "Racha",
	DaysSuffix:    "días",
	LearnedSuffix: "tarjetas aprendidas",

	ChoiceHelp:   "1-4 elegir · ↑/↓ mover · ENTER confirmar · ESC menú",
	ContinueHelp: "ENTER continuar · ESC menú",
	RestartHelp:  "ENTER reiniciar · ESC menú",
	BackHelp:     "ESC volver al menú",
	SessionDone:  "¡Sesión completada!",
	ScoreLabel:   "Aciertos",

	KanaTitle:       "Entrenador de Kana",
	KanaPrompt:      "¿Cómo se lee?",
	KanaGroupAll:    "Todo",
	KanaPickHelp:    "↑/↓ moverse · ENTER empezar · ESC volver",
	KanaFluent:      "fluido",
	KanaMasteredFmt: "%d/%d",
	KanaLockedHint:  "Domina el hiragana primero para desbloquear el katakana.",
	FluentBadge:     "¡Lectura fluida! Ya puedes leer palabras y frases.",

	KanaChartTitle: "Tabla de Kana",
	KanaChartHelp:  "← → cambiar página · ESC volver",
	KanaBasic:      "Básico",
	KanaVoiced:     "Dakuten / Handakuten",
	KanaCombo:      "Combinaciones",

	QuizTitle:       "Quiz",
	QuizQuestionFmt: "¿Cómo se dice \"%s\" en japonés?",
	ReviewLabel:     "Repasa",

	FlashTitle:        "Flashcards",
	ReviewScreenTitle: "Repaso",
	RevealHelp:        "ESPACIO revelar · ESC menú",
	GradePrompt:       "¿Qué tal lo recordaste?",
	GradeAgain:        "Otra vez",
	GradeHard:         "Difícil",
	GradeGood:         "Bien",
	GradeEasy:         "Fácil",
	ReviewedLabel:     "Tarjetas repasadas",
	NothingDue:        "No hay tarjetas para repasar ahora. Vuelve más tarde.",
	Today:             "hoy",
	DayShort:          "d",

	StatsTitle:    "Mis estadísticas",
	BestLabel:     "récord",
	HiraganaLabel: "Hiragana",
	KatakanaLabel: "Katakana",

	WelcomeTitle:  "Bienvenido a Polyglot",
	WelcomeIntro:  "Vas a aprender japonés desde el español.",
	ControlsTitle: "Controles básicos:",
	ControlsKeys: []string{
		"↑ ↓      moverte por las opciones",
		"ENTER    confirmar",
		"ESPACIO  revelar respuesta (en flashcards)",
		"ESC      volver al menú",
		"Q        salir",
	},
	WelcomeNext:     "ENTER  probemos un ejercicio →",
	PracticeTitle:   "Práctica guiada",
	SampleWord:      "みず",
	SampleRomaji:    "mizu",
	SamplePrompt:    "¿Qué significa esta palabra?",
	SampleOptions:   []string{"Fuego", "Agua", "Gato", "Árbol"},
	SampleCorrect:   1,
	SampleHint:      "◀ pista: ¡es esta!",
	PracticeCorrect: "¡Genial! Ya sabes lo básico.",
	PracticeRetry:   "Casi… la respuesta correcta está marcada. Inténtalo.",
	PracticeNext:    "ENTER  continuar →",
	DoneTitle:       "¡Todo listo!",
	DoneRecommend:   "Te recomiendo empezar por el Entrenador de Kana.",
	DoneNext:        "ENTER  ir al menú principal",
}

// Default is the active UI language.
var Default = ES
