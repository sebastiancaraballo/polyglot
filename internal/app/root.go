package app

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/content"
	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/nav"
	"github.com/sebastiancaraballo/polyglot/internal/review"
	"github.com/sebastiancaraballo/polyglot/internal/screens/flashcards"
	"github.com/sebastiancaraballo/polyglot/internal/screens/kana"
	"github.com/sebastiancaraballo/polyglot/internal/screens/kanachart"
	"github.com/sebastiancaraballo/polyglot/internal/screens/menu"
	"github.com/sebastiancaraballo/polyglot/internal/screens/onboarding"
	"github.com/sebastiancaraballo/polyglot/internal/screens/profiles"
	"github.com/sebastiancaraballo/polyglot/internal/screens/profilesetup"
	"github.com/sebastiancaraballo/polyglot/internal/screens/quiz"
	"github.com/sebastiancaraballo/polyglot/internal/screens/settings"
	"github.com/sebastiancaraballo/polyglot/internal/screens/stats"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/study"
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
	case nav.SetShowRomajiMsg:
		return m.setShowRomaji(msg.Enabled)
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

// setShowRomaji persists the active profile's romaji preference and updates the
// shared context so subsequently-built screens pick it up. The settings screen
// stays mounted; it already reflects the new value locally.
func (m rootModel) setShowRomaji(enabled bool) (tea.Model, tea.Cmd) {
	if m.ctx.profile.ID != 0 {
		if err := m.ctx.store.SetShowRomaji(context.Background(), m.ctx.profile.ID, enabled); err != nil {
			return m, tea.Quit
		}
	}
	m.ctx.profile.ShowRomaji = enabled
	return m, nil
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
	case nav.KanaChart:
		return kanachart.New(kanachart.Deps{
			Theme: m.ctx.theme, Msgs: m.ctx.msgs, Kana: m.ctx.course.Kana,
		})
	case nav.Flashcards:
		dec := m.ctx.decoder()
		return flashcards.New(flashcards.Deps{
			Theme: m.ctx.theme, Msgs: m.ctx.msgs, Store: m.ctx.store,
			ProfileID:  m.ctx.profile.ID,
			Items:      decodableItems(review.VocabItems(m.ctx.course.Lessons), dec),
			ShowRomaji: m.ctx.profile.ShowRomaji, Title: m.ctx.msgs.FlashTitle,
		})
	case nav.Review:
		dec := m.ctx.decoder()
		return flashcards.New(flashcards.Deps{
			Theme: m.ctx.theme, Msgs: m.ctx.msgs, Store: m.ctx.store,
			ProfileID: m.ctx.profile.ID, Items: decodableItems(m.ctx.reviewItems(), dec),
			ShowRomaji: m.ctx.profile.ShowRomaji, Title: m.ctx.msgs.ReviewScreenTitle,
		})
	case nav.Quiz:
		dec := m.ctx.decoder()
		return quiz.New(quiz.Deps{
			Theme: m.ctx.theme, Msgs: m.ctx.msgs, Store: m.ctx.store,
			ProfileID: m.ctx.profile.ID, Cards: decodableCards(m.ctx.allCards(), dec),
			ShowRomaji: m.ctx.profile.ShowRomaji,
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
		return settings.New(settings.Deps{
			Theme: m.ctx.theme, Msgs: m.ctx.msgs, ShowRomaji: m.ctx.profile.ShowRomaji,
		})
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

// decoder builds the progressive reading decoder from the active profile's kana
// progress. It is best-effort: on a storage error the decoder simply treats no
// kana as mastered (nothing decodable yet).
func (c appContext) decoder() study.Decoder {
	progress, _ := c.store.GetKanaProgress(context.Background(), c.profile.ID)
	return study.NewDecoder(c.course.Kana, progress)
}

// decodableItems keeps every non-vocabulary item (e.g. kana) and only those
// vocabulary items the learner can already read with their mastered kana.
func decodableItems(items []review.Item, dec study.Decoder) []review.Item {
	out := make([]review.Item, 0, len(items))
	for _, it := range items {
		if it.Strand == review.Vocab && !dec.Decodable(it.Answer) {
			continue
		}
		out = append(out, it)
	}
	return out
}

// decodableCards keeps only the cards the learner can already read.
func decodableCards(cards []model.Card, dec study.Decoder) []model.Card {
	out := make([]model.Card, 0, len(cards))
	for _, c := range cards {
		if dec.Decodable(c.JP) {
			out = append(out, c)
		}
	}
	return out
}

// allCards flattens every lesson's cards into a single slice.
func (c appContext) allCards() []model.Card {
	var cards []model.Card
	for _, lesson := range c.course.Lessons {
		cards = append(cards, lesson.Cards...)
	}
	return cards
}

// reviewItems is the full cross-curriculum set of schedulable items: vocabulary
// and kana, interleaved by the review queue.
func (c appContext) reviewItems() []review.Item {
	items := review.VocabItems(c.course.Lessons)
	return append(items, review.KanaItems(c.course.Kana)...)
}

// reviewableTotal is the number of distinct items that can enter the review queue
// (vocabulary cards plus kana). It keeps the learned/total progress badge coherent
// now that kana participates in spaced repetition.
func (c appContext) reviewableTotal() int {
	return len(c.allCards()) + len(c.course.Kana)
}

// summary computes the menu's progress badge. It is best-effort: on a storage
// error it falls back to zero values rather than failing navigation.
func (c appContext) summary() menu.Summary {
	ctx := context.Background()

	stats, _ := c.store.GetStats(ctx, c.profile.ID)
	learned, _ := c.store.CountLearnedCards(ctx, c.profile.ID)

	// Reading is locked only while nothing is decodable yet; once the learner has
	// mastered enough kana to read at least one card, the reading activities open
	// and show that growing decodable subset.
	readable := len(decodableCards(c.allCards(), c.decoder()))

	return menu.Summary{
		Name:          c.profile.Name,
		XP:            stats.XP,
		Streak:        stats.Streak,
		Learned:       learned,
		Total:         c.reviewableTotal(),
		ReadingLocked: readable == 0,
	}
}
