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
	KanaTitle         string
	KanaPrompt        string
	KanaGroupAll      string
	KanaPickHelp      string
	KanaFluent        string // badge on a fully-mastered group
	KanaMasteredFmt   string // "%d/%d" mastered count
	KanaUnlockHintFmt string // why a katakana group is locked, with live "%d/%d" hiragana progress
	KanaMasteryNote   string // explains what "mastered" means, shown under the picker
	FluentBadge       string // syllabary-fluency badge on the summary screen

	// Kana trainer first-time intro
	KanaIntroTitle string
	KanaIntroBody  string // explains the hiragana → katakana → reading path and mastery
	KanaIntroHelp  string // dismiss help

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

	// Rikai (grammar-pattern practice)
	ItemRikai          string
	RikaiTitle         string
	RikaiLocked        string // menu lock notice / empty-state message
	RikaiPickHelp      string
	RikaiMasteryNote   string // explains one-variable-at-a-time, shown under the picker
	RikaiUnlockHint    string // why the cursor's pattern is locked
	RikaiQuestionFmt   string // cue for the blank, "¿Cómo se dice \"%s\" aquí?"
	RikaiPatternFluent string // badge on a fully-mastered pattern
	RikaiMasteredFmt   string // "%d/%d" slots mastered

	// Story (Katsudoo)
	ItemStory          string
	StoryTitle         string
	StoryPickHelp      string
	StoryProgressFmt   string // "%d/%d" beats seen, e.g. "2/6 escenas"
	StoryCompleteBadge string // badge on a finished chapter
	StoryEmpty         string // shown when no chapters exist yet
	StoryDoneTitle     string // chapter-end screen title
	StoryDoneNext      string // chapter-end help line

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

	KanaTitle:         "Entrenador de Kana",
	KanaPrompt:        "¿Cómo se lee?",
	KanaGroupAll:      "Todo",
	KanaPickHelp:      "↑/↓ moverse · ENTER empezar · ESC volver",
	KanaFluent:        "fluido",
	KanaMasteredFmt:   "%d/%d",
	KanaUnlockHintFmt: "Domina el hiragana para desbloquear el katakana — %d/%d.",
	KanaMasteryNote:   "Dominar = responder bien varias veces seguidas.",
	FluentBadge:       "¡Kana fluido! Ya puedes leer todas las palabras y frases.",

	KanaIntroTitle: "Entrenador de Kana",
	KanaIntroBody: "El kana es la base para leer japonés. Lo aprenderás en este orden:\n\n" +
		"1. Hiragana\n" +
		"2. Katakana\n" +
		"3. Lectura de palabras y frases\n\n" +
		"Cada etapa se desbloquea al dominar la anterior. Dominas un kana cuando lo " +
		"reconoces bien varias veces seguidas: así afianzas la lectura antes de " +
		"pasar a leer.",
	KanaIntroHelp: "ENTER empezar · ESC volver",

	KanaChartTitle: "Tabla de Kana",
	KanaChartHelp:  "← → cambiar página · ESC volver",
	KanaBasic:      "Básico",
	KanaVoiced:     "Dakuten / Handakuten",
	KanaCombo:      "Combinaciones",

	QuizTitle:       "Quiz",
	QuizQuestionFmt: "¿Cómo se dice \"%s\" en japonés?",
	ReviewLabel:     "Repasa",

	ItemRikai:          "Rikai (gramática)",
	RikaiTitle:         "Rikai",
	RikaiLocked:        "Aprende más vocabulario para desbloquear Rikai.",
	RikaiPickHelp:      "↑/↓ moverse · ENTER empezar · ESC volver",
	RikaiMasteryNote:   "Cada ronda cambia una sola palabra del patrón; el resto queda fija.",
	RikaiUnlockHint:    "Aprende más palabras de este patrón primero.",
	RikaiQuestionFmt:   "¿Cómo se dice \"%s\" aquí?",
	RikaiPatternFluent: "dominado",
	RikaiMasteredFmt:   "%d/%d",

	ItemStory:          "Katsudoo (historia)",
	StoryTitle:         "Katsudoo",
	StoryPickHelp:      "↑/↓ moverse · ENTER empezar · ESC volver",
	StoryProgressFmt:   "%d/%d escenas",
	StoryCompleteBadge: "✓ completado",
	StoryEmpty:         "Aún no hay capítulos disponibles.",
	StoryDoneTitle:     "Capítulo completado",
	StoryDoneNext:      "ENTER volver a los capítulos",

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
