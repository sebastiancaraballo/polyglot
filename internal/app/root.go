package app

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/content"
	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/screens/flashcards"
	"github.com/sebastiancaraballo/polyglot/internal/screens/kana"
	"github.com/sebastiancaraballo/polyglot/internal/screens/menu"
	"github.com/sebastiancaraballo/polyglot/internal/screens/onboarding"
	"github.com/sebastiancaraballo/polyglot/internal/screens/profiles"
	"github.com/sebastiancaraballo/polyglot/internal/screens/profilesetup"
	"github.com/sebastiancaraballo/polyglot/internal/screens/quiz"
	"github.com/sebastiancaraballo/polyglot/internal/screens/settings"
	"github.com/sebastiancaraballo/polyglot/internal/screens/stats"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

// appContext carries the shared dependencies needed to build any screen.
type appContext struct {
	store   storage.Storage
	course  *content.Course
	profile model.Profile
	theme   ui.Theme
	msgs    i18n.Messages
	version string
	dbPath  string
}

// rootModel is the top-level Bubble Tea model. It tracks terminal size and
// routes between screens in response to navigation messages.
type rootModel struct {
	ctx           appContext
	screen        tea.Model
	setupTutorial bool
	width         int
	height        int
}

func newRoot(ctx appContext) rootModel {
	m := rootModel{ctx: ctx}
	initial := nav.Menu
	switch {
	case ctx.profile.ID == 0:
		initial = nav.ProfileSetup
		m.setupTutorial = true
	case !ctx.profile.Onboarded:
		initial = nav.Onboarding
	}
	m.screen = m.build(initial)
	return m
}

func (m rootModel) Init() tea.Cmd {
	return m.screen.Init()
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case nav.GoToMsg:
		return m.route(msg.Screen)
	case nav.BackMsg:
		if m.ctx.profile.ID == 0 {
			m.setupTutorial = true
			return m.route(nav.ProfileSetup)
		}
		return m.route(nav.Menu)
	case nav.ProfileCreatedMsg:
		return m.profileCreated(msg)
	case nav.SwitchProfileMsg:
		return m.switchProfile(msg.ID)
	case nav.CreateProfileMsg:
		m.setupTutorial = false
		return m.route(nav.ProfileSetup)
	case nav.DeleteProfileMsg:
		return m.deleteActiveProfile()
	case nav.WipeDataMsg:
		return m.wipeAndReset()
	}
	var cmd tea.Cmd
	m.screen, cmd = m.screen.Update(msg)
	return m, cmd
}

func (m rootModel) View() tea.View {
	return m.screen.View()
}

// wipeAndReset deletes all local data and returns to first-run profile setup. It
// owns this flow because a leaf screen cannot close and reopen the shared storage
// connection. On a fatal re-open failure it quits, since the application can no
// longer persist anything.
func (m rootModel) wipeAndReset() (tea.Model, tea.Cmd) {
	_ = m.ctx.store.Close()
	if err := storage.Remove(m.ctx.dbPath); err != nil {
		return m, tea.Quit
	}
	store, err := storage.Open(m.ctx.dbPath)
	if err != nil {
		return m, tea.Quit
	}
	m.ctx.store = store
	m.ctx.profile = model.Profile{}
	m.setupTutorial = true
	return m.route(nav.ProfileSetup)
}

func (m rootModel) profileCreated(msg nav.ProfileCreatedMsg) (tea.Model, tea.Cmd) {
	ctx := context.Background()
	if err := m.ctx.store.SetActiveProfileID(ctx, msg.ID); err != nil {
		return m, tea.Quit
	}
	profile, err := m.ctx.store.GetProfile(ctx, msg.ID)
	if err != nil {
		return m, tea.Quit
	}
	if !msg.Tutorial {
		if err := m.ctx.store.SetOnboarded(ctx, msg.ID); err != nil {
			return m, tea.Quit
		}
		profile.Onboarded = true
	}
	m.ctx.profile = profile
	m.setupTutorial = false
	if msg.Tutorial {
		return m.route(nav.Onboarding)
	}
	return m.route(nav.Menu)
}

func (m rootModel) switchProfile(id int64) (tea.Model, tea.Cmd) {
	ctx := context.Background()
	if err := m.ctx.store.SetActiveProfileID(ctx, id); err != nil {
		return m, tea.Quit
	}
	profile, err := m.ctx.store.GetProfile(ctx, id)
	if err != nil {
		return m, tea.Quit
	}
	m.ctx.profile = profile
	m.setupTutorial = false
	return m.route(m.screenForProfile(profile))
}

func (m rootModel) deleteActiveProfile() (tea.Model, tea.Cmd) {
	ctx := context.Background()
	if m.ctx.profile.ID != 0 {
		if err := m.ctx.store.DeleteProfile(ctx, m.ctx.profile.ID); err != nil && !errors.Is(err, storage.ErrNotFound) {
			return m, tea.Quit
		}
	}
	remaining, err := m.ctx.store.ListProfiles(ctx)
	if err != nil {
		return m, tea.Quit
	}
	if len(remaining) == 0 {
		m.ctx.profile = model.Profile{}
		m.setupTutorial = true
		return m.route(nav.ProfileSetup)
	}
	next := remaining[0]
	if err := m.ctx.store.SetActiveProfileID(ctx, next.ID); err != nil {
		return m, tea.Quit
	}
	m.ctx.profile = next
	m.setupTutorial = false
	return m.route(m.screenForProfile(next))
}

// route switches to a new screen, seeding it with the current terminal size.
func (m rootModel) route(s nav.Screen) (tea.Model, tea.Cmd) {
	m.screen = m.build(s)
	sized, cmd := m.screen.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	m.screen = sized
	return m, tea.Batch(m.screen.Init(), cmd)
}

// build constructs the model for a screen using the shared context.
func (m rootModel) build(s nav.Screen) tea.Model {
	switch s {
	case nav.Kana:
		return kana.New(kana.Deps{
			Theme: m.ctx.theme, Msgs: m.ctx.msgs, Store: m.ctx.store,
			ProfileID: m.ctx.profile.ID, Items: m.ctx.course.Kana,
		})
	case nav.Flashcards:
		return flashcards.New(flashcards.Deps{
			Theme: m.ctx.theme, Msgs: m.ctx.msgs, Store: m.ctx.store,
			ProfileID: m.ctx.profile.ID, Cards: m.ctx.allCards(),
		})
	case nav.Quiz:
		return quiz.New(quiz.Deps{
			Theme: m.ctx.theme, Msgs: m.ctx.msgs, Store: m.ctx.store,
			ProfileID: m.ctx.profile.ID, Cards: m.ctx.allCards(),
		})
	case nav.Stats:
		return stats.New(stats.Deps{
			Theme: m.ctx.theme, Msgs: m.ctx.msgs, Store: m.ctx.store,
			ProfileID: m.ctx.profile.ID, Course: m.ctx.course,
		})
	case nav.Onboarding:
		return onboarding.New(onboarding.Deps{
			Theme: m.ctx.theme, Msgs: m.ctx.msgs, Store: m.ctx.store,
			ProfileID: m.ctx.profile.ID,
		})
	case nav.Settings:
		return settings.New(settings.Deps{Theme: m.ctx.theme, Msgs: m.ctx.msgs})
	case nav.ProfileSetup:
		return profilesetup.New(profilesetup.Deps{
			Theme: m.ctx.theme, Msgs: m.ctx.msgs, Store: m.ctx.store,
			Tutorial: m.setupTutorial,
		})
	case nav.Profiles:
		return profiles.New(profiles.Deps{
			Theme: m.ctx.theme, Msgs: m.ctx.msgs, Store: m.ctx.store,
			ActiveID: m.ctx.profile.ID,
		})
	default:
		return menu.New(m.ctx.theme, m.ctx.msgs, m.ctx.summary(), m.ctx.version)
	}
}

func (m rootModel) screenForProfile(profile model.Profile) nav.Screen {
	if profile.ID == 0 {
		return nav.ProfileSetup
	}
	if !profile.Onboarded {
		return nav.Onboarding
	}
	return nav.Menu
}

// allCards flattens every lesson's cards into a single slice.
func (c appContext) allCards() []model.Card {
	var cards []model.Card
	for _, lesson := range c.course.Lessons {
		cards = append(cards, lesson.Cards...)
	}
	return cards
}

// summary computes the menu's progress badge. It is best-effort: on a storage
// error it falls back to zero values rather than failing navigation.
func (c appContext) summary() menu.Summary {
	ctx := context.Background()

	stats, _ := c.store.GetStats(ctx, c.profile.ID)
	learned, _ := c.store.CountLearnedCards(ctx, c.profile.ID)

	return menu.Summary{
		Name:    c.profile.Name,
		XP:      stats.XP,
		Streak:  stats.Streak,
		Learned: learned,
		Total:   len(c.allCards()),
	}
}
