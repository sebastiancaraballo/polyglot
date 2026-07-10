//! The application router and terminal event loop.
//!
//! Port of the Go `internal/nav` + `internal/app/root` wiring. Screens are a
//! stack: the menu is the root, activities are pushed on top, and going back
//! pops. (During the TUI port, not-yet-ported destinations render a placeholder
//! screen so navigation is exercisable end-to-end.)

use polyglot_core::i18n::Messages;
use ratatui::crossterm::event::{self, Event, KeyCode, KeyEventKind, KeyModifiers};
use ratatui::Frame;

use crate::frame::draw_frame;
use crate::screens::menu::{Menu, Summary};
use crate::screens::placeholder::Placeholder;
use crate::theme::Theme;

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
    /// Stay on the current screen.
    Stay,
    /// Quit the application.
    Quit,
    /// Push a new screen for `Dest`.
    Push(Dest),
    /// Pop back to the previous screen.
    Pop,
}

/// Whether the event loop should continue or exit.
#[derive(PartialEq, Eq)]
enum Flow {
    Continue,
    Quit,
}

enum Screen {
    Menu(Menu),
    Placeholder(Placeholder),
}

/// The running application.
pub struct App {
    theme: Theme,
    msgs: &'static Messages,
    stack: Vec<Screen>,
}

impl App {
    /// Builds the app rooted at the main menu.
    pub fn new(theme: Theme, msgs: &'static Messages, summary: Summary, version: String) -> App {
        App {
            theme,
            msgs,
            stack: vec![Screen::Menu(Menu::new(msgs, summary, version))],
        }
    }

    fn render(&self, f: &mut Frame) {
        let inner = draw_frame(f, &self.theme);
        match self.stack.last().expect("stack is never empty") {
            Screen::Menu(m) => m.render(f, inner, &self.theme, self.msgs),
            Screen::Placeholder(p) => p.render(f, inner, &self.theme),
        }
    }

    fn handle(&mut self, code: KeyCode, mods: KeyModifiers) -> Flow {
        let transition = match self.stack.last_mut().expect("stack is never empty") {
            Screen::Menu(m) => m.handle(code, mods),
            Screen::Placeholder(p) => p.handle(code, mods),
        };
        match transition {
            Transition::Stay => Flow::Continue,
            Transition::Quit => Flow::Quit,
            Transition::Push(dest) => {
                self.stack.push(Screen::Placeholder(Placeholder::new(dest)));
                Flow::Continue
            }
            Transition::Pop => {
                if self.stack.len() > 1 {
                    self.stack.pop();
                }
                Flow::Continue
            }
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
