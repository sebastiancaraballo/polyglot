//! The application router and terminal event loop.
//!
//! Port of the Go `internal/nav` + `internal/app/root` wiring. Screens are a
//! stack: the menu is the root, activities are pushed on top, and going back
//! pops (refreshing the menu's progress summary). Not-yet-ported destinations
//! render a placeholder so navigation is exercisable end-to-end.

use polyglot_core::content::Course;
use polyglot_core::i18n::Messages;
use polyglot_core::model::Profile;
use polyglot_core::storage::{SqliteStore, StorageError};
use ratatui::crossterm::event::{self, Event, KeyCode, KeyEventKind, KeyModifiers};
use ratatui::Frame;

use polyglot_core::review;

use crate::frame::draw_frame;
use crate::screens::flashcards::Flashcards;
use crate::screens::kana::KanaTrainer;
use crate::screens::kanachart::KanaChart;
use crate::screens::menu::{Menu, Summary};
use crate::screens::placeholder::Placeholder;
use crate::screens::quiz::Quiz;
use crate::screens::stats::Stats;
use crate::theme::Theme;

/// The shared context handed to a screen's `handle`, so interactive screens can
/// persist progress. Read-only screens ignore it.
pub struct Ctx<'a> {
    pub store: &'a SqliteStore,
    pub profile_id: Option<i64>,
}

/// A top-level navigation destination (mirrors the Go `nav.Screen`).
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Dest {
    Kana,
    Flashcards,
    Review,
    Quiz,
    Stats,
    Settings,
    Profiles,
    KanaChart,
    Rikai,
    Story,
    Assessment,
}

impl Dest {
    /// A human-readable title, used by the placeholder screen until the real
    /// screen is ported.
    pub fn title(self) -> &'static str {
        match self {
            Dest::Kana => "Entrenador de Kana",
            Dest::Flashcards => "Flashcards",
            Dest::Review => "Repaso",
            Dest::Quiz => "Quiz",
            Dest::Stats => "Mis estadísticas",
            Dest::Settings => "Ajustes",
            Dest::Profiles => "Perfiles",
            Dest::KanaChart => "Tabla de Kana",
            Dest::Rikai => "Rikai",
            Dest::Story => "Katsudoo",
            Dest::Assessment => "Examen N5",
        }
    }
}

/// The outcome of a key press within a screen.
pub enum Transition {
    Stay,
    Quit,
    Push(Dest),
    Pop,
}

#[derive(PartialEq, Eq)]
enum Flow {
    Continue,
    Quit,
}

enum Screen {
    Menu(Box<Menu>),
    Stats(Stats),
    KanaChart(Box<KanaChart>),
    Kana(Box<KanaTrainer>),
    Flashcards(Box<Flashcards>),
    Quiz(Box<Quiz>),
    Placeholder(Placeholder),
}

/// The running application: shared context (storage, content, active profile)
/// plus the screen stack.
pub struct App {
    theme: Theme,
    msgs: &'static Messages,
    version: String,
    store: SqliteStore,
    course: Course,
    profile_id: Option<i64>,
    stack: Vec<Screen>,
}

impl App {
    /// Builds the app rooted at the main menu, resolving the active profile and
    /// computing the menu's progress summary from storage.
    pub fn new(
        theme: Theme,
        msgs: &'static Messages,
        version: String,
        store: SqliteStore,
        course: Course,
    ) -> Result<App, StorageError> {
        let profile_id = resolve_profile(&store)?.map(|p| p.id);
        let summary = Summary::build(&store, &course, profile_id)?;
        let menu = Menu::new(msgs, summary, version.clone());
        Ok(App {
            theme,
            msgs,
            version,
            store,
            course,
            profile_id,
            stack: vec![Screen::Menu(Box::new(menu))],
        })
    }

    fn render(&self, f: &mut Frame) {
        let inner = draw_frame(f, &self.theme);
        match self.stack.last().expect("stack is never empty") {
            Screen::Menu(s) => s.render(f, inner, &self.theme, self.msgs),
            Screen::Stats(s) => s.render(f, inner, &self.theme, self.msgs),
            Screen::KanaChart(s) => s.render(f, inner, &self.theme, self.msgs),
            Screen::Kana(s) => s.render(f, inner, &self.theme, self.msgs),
            Screen::Flashcards(s) => s.render(f, inner, &self.theme, self.msgs),
            Screen::Quiz(s) => s.render(f, inner, &self.theme, self.msgs),
            Screen::Placeholder(s) => s.render(f, inner, &self.theme),
        }
    }

    fn handle(&mut self, code: KeyCode, mods: KeyModifiers) -> Flow {
        let transition = {
            let ctx = Ctx {
                store: &self.store,
                profile_id: self.profile_id,
            };
            match self.stack.last_mut().expect("stack is never empty") {
                Screen::Menu(s) => s.handle(code, mods, &ctx),
                Screen::Stats(s) => s.handle(code, mods, &ctx),
                Screen::KanaChart(s) => s.handle(code, mods, &ctx),
                Screen::Kana(s) => s.handle(code, mods, &ctx),
                Screen::Flashcards(s) => s.handle(code, mods, &ctx),
                Screen::Quiz(s) => s.handle(code, mods, &ctx),
                Screen::Placeholder(s) => s.handle(code, mods, &ctx),
            }
        };
        match transition {
            Transition::Stay => Flow::Continue,
            Transition::Quit => Flow::Quit,
            Transition::Push(dest) => {
                let screen = self.build_screen(dest);
                self.stack.push(screen);
                Flow::Continue
            }
            Transition::Pop => {
                if self.stack.len() > 1 {
                    self.stack.pop();
                }
                self.refresh_menu();
                Flow::Continue
            }
        }
    }

    /// Builds the screen for a destination, reading its data from the shared
    /// context. Unported destinations fall back to a placeholder.
    fn build_screen(&self, dest: Dest) -> Screen {
        match dest {
            Dest::Stats => Screen::Stats(Stats::new(&self.store, &self.course, self.profile_id)),
            Dest::KanaChart => Screen::KanaChart(Box::new(KanaChart::new(&self.course))),
            Dest::Kana => Screen::Kana(Box::new(KanaTrainer::new(
                &self.store,
                &self.course,
                self.profile_id,
            ))),
            Dest::Flashcards => Screen::Flashcards(Box::new(Flashcards::new(
                &self.store,
                self.profile_id,
                &review::vocab_items(&self.course.lessons),
                self.msgs.flash_title.clone(),
                self.show_romaji(),
            ))),
            Dest::Review => {
                let mut items = review::vocab_items(&self.course.lessons);
                items.extend(review::kana_items(&self.course.kana));
                Screen::Flashcards(Box::new(Flashcards::new(
                    &self.store,
                    self.profile_id,
                    &items,
                    self.msgs.review_screen_title.clone(),
                    self.show_romaji(),
                )))
            }
            Dest::Quiz => {
                let cards: Vec<_> = self
                    .course
                    .lessons
                    .iter()
                    .flat_map(|l| l.cards.iter().cloned())
                    .collect();
                Screen::Quiz(Box::new(Quiz::new(cards, self.show_romaji())))
            }
            other => Screen::Placeholder(Placeholder::new(other)),
        }
    }

    /// The active profile's romaji preference (default on when unknown).
    fn show_romaji(&self) -> bool {
        self.profile_id
            .and_then(|id| self.store.get_profile(id).ok())
            .is_none_or(|p| p.show_romaji)
    }

    /// Rebuilds the menu's progress summary after returning to it, so gating and
    /// counters reflect any progress made in the popped screen.
    fn refresh_menu(&mut self) {
        if self.stack.len() != 1 {
            return;
        }
        if let Ok(summary) = Summary::build(&self.store, &self.course, self.profile_id) {
            self.stack[0] = Screen::Menu(Box::new(Menu::new(
                self.msgs,
                summary,
                self.version.clone(),
            )));
        }
    }
}

/// Returns the persisted active profile, the first existing profile, or `None`
/// on a first run with no profiles yet.
fn resolve_profile(store: &SqliteStore) -> Result<Option<Profile>, StorageError> {
    if let Some(id) = store.active_profile_id()? {
        match store.get_profile(id) {
            Ok(p) => return Ok(Some(p)),
            Err(StorageError::NotFound) => {}
            Err(e) => return Err(e),
        }
    }
    let profiles = store.list_profiles()?;
    match profiles.into_iter().next() {
        None => Ok(None),
        Some(p) => {
            store.set_active_profile_id(p.id)?;
            Ok(Some(p))
        }
    }
}

/// Runs the terminal event loop until the user quits.
pub fn run(mut app: App) -> std::io::Result<()> {
    let mut terminal = ratatui::init();
    let result = loop {
        if let Err(e) = terminal.draw(|f| app.render(f)) {
            break Err(e);
        }
        match event::read() {
            Ok(Event::Key(key)) if key.kind == KeyEventKind::Press => {
                if app.handle(key.code, key.modifiers) == Flow::Quit {
                    break Ok(());
                }
            }
            Ok(_) => {}
            Err(e) => break Err(e),
        }
    };
    ratatui::restore();
    result
}
