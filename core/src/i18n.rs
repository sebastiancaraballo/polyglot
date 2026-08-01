//! User-facing strings for one UI language.
//!
//! Port of the Go `internal/i18n` package. Keeping every string in one struct
//! lets additional UI languages be added without touching screens. Format
//! placeholders keep the Go `%s`/`%d` spelling verbatim (they are data); the
//! TUI layer interpolates them.
//!
//! The [`messages!`] macro defines the [`Messages`] struct, the Spanish
//! constructor, and [`Messages::all_strings`] from a single field list, so the
//! content guards (no banned emoji, uppercase key labels) can iterate every
//! string without hand-maintaining a parallel list.

use std::sync::LazyLock;

macro_rules! messages {
    (
        strings { $($sfield:ident : $sval:expr),* $(,)? }
        lists { $($lfield:ident : [$($lval:expr),* $(,)?]),* $(,)? }
        ints { $($ifield:ident : $ival:expr),* $(,)? }
    ) => {
        /// Every user-facing string for one UI language.
        #[derive(Clone, Debug)]
        pub struct Messages {
            $(pub $sfield: String,)*
            $(pub $lfield: Vec<String>,)*
            $(pub $ifield: i64,)*
        }

        impl Messages {
            /// The Spanish localization, used by default in v1.
            pub fn es() -> Messages {
                Messages {
                    $($sfield: $sval.to_string(),)*
                    $($lfield: vec![$($lval.to_string()),*],)*
                    $($ifield: $ival,)*
                }
            }

            /// Every string field (and every element of every list field),
            /// for content-guard tests.
            #[allow(clippy::vec_init_then_push)]
            pub fn all_strings(&self) -> Vec<&str> {
                let mut v: Vec<&str> = Vec::new();
                $(v.push(self.$sfield.as_str());)*
                $(for s in &self.$lfield { v.push(s.as_str()); })*
                v
            }
        }
    };
}

messages! {
    strings {
        app_name: "Polyglot",
        tagline: "es → ja",

        item_kana: "Entrenador de Kana",
        item_kana_chart: "Tabla de Kana",
        item_flashcards: "Flashcards",
        item_review: "Repaso",
        item_quiz: "Quiz de opción múltiple",
        item_stats: "Mis estadísticas",
        item_settings: "Ajustes",
        item_quit: "Salir",
        switch_profile: "Cambiar de perfil",
        menu_help: "↑/↓ moverse · ENTER entrar/cambiar perfil · Q salir",
        menu_help_sub: "↑/↓ moverse · ENTER elegir · ESC volver",
        cat_learn: "Aprender",
        cat_read: "Leer",
        cat_evaluate: "Evaluar",
        cat_tools: "Herramientas",
        reading_locked: "Aprende a leer los kana con fluidez primero.",

        settings_title: "Ajustes",
        settings_help: "↑/↓ moverse · ENTER cambiar/confirmar · ESC volver",
        show_romaji_label: "Mostrar romaji",
        option_on: "Sí",
        option_off: "No",
        delete_profile: "Borrar mi perfil",
        delete_profile_warning: "Esto borra solo el perfil actual y su progreso. No se puede deshacer.",
        confirm_delete_profile: "Sí, borrar mi perfil",
        delete_all_data: "Borrar todos los datos",
        delete_all_warning: "Esto borra todos los perfiles, progreso y estadísticas. No se puede deshacer.",
        confirm_delete: "Sí, borrar todo",
        cancel_label: "Cancelar",
        confirm_help: "↑/↓ elegir · ENTER confirmar · ESC cancelar",

        profile_name_title: "Crea tu perfil",
        profile_name_prompt: "¿Cómo te llamas?",
        profile_name_placeholder: "Tu nombre",
        profile_name_empty: "Escribe un nombre.",
        profile_name_too_long_fmt: "Máximo %d caracteres.",
        profile_name_invalid: "Usa letras, espacios o puntuación de nombre.",
        profile_name_help_first: "Escribe tu nombre · ENTER crear perfil",
        profile_name_help_cancel: "Escribe tu nombre · ENTER crear perfil · ESC cancelar",
        profile_create_error: "No pude crear el perfil.",
        profiles_title: "Perfiles",
        profile_create_new: "＋ Crear nuevo perfil",
        active_profile_label: "actual",
        profiles_help: "↑/↓ mover · ENTER elegir · ESC menú",
        no_profiles: "No hay perfiles todavía.",

        xp_label: "XP",
        streak_label: "Racha",
        days_suffix: "días",
        learned_suffix: "tarjetas aprendidas",

        choice_help: "1-4 elegir · ↑/↓ mover · ENTER confirmar · ESC menú",
        continue_help: "ENTER continuar · ESC menú",
        restart_help: "ENTER reiniciar · ESC menú",
        back_help: "ESC volver al menú",
        session_done: "¡Sesión completada!",
        score_label: "Aciertos",

        kana_title: "Entrenador de Kana",
        kana_prompt: "¿Cómo se lee?",
        kana_prompt_reverse: "¿Qué kana es?",
        kana_group_all: "Todo",
        kana_pick_help: "↑/↓ grupo · ←/→ dirección · ENTER empezar · ESC volver",
        kana_direction_fmt: "Dirección: %s",
        kana_dir_forward: "kana → romaji",
        kana_dir_reverse: "romaji → kana",
        kana_fluent: "fluido",
        kana_mastered_fmt: "%d/%d",
        kana_unlock_hint_fmt: "Domina el hiragana para desbloquear el katakana — %d/%d.",
        kana_mastery_note: "Dominar = responder bien varias veces seguidas.",
        fluent_badge: "¡Kana fluido! Ya puedes leer todas las palabras y frases.",

        kana_intro_title: "Entrenador de Kana",
        kana_intro_body: "El kana es la base para leer japonés. Lo aprenderás en este orden:\n\n1. Hiragana\n2. Katakana\n3. Lectura de palabras y frases\n\nCada etapa se desbloquea al dominar la anterior. Dominas un kana cuando lo reconoces bien varias veces seguidas: así afianzas la lectura antes de pasar a leer.",
        kana_intro_help: "ENTER empezar · ESC volver",

        kana_chart_title: "Tabla de Kana",
        kana_chart_help: "← → cambiar página · ESC volver",
        kana_basic: "Básico",
        kana_voiced: "Dakuten / Handakuten",
        kana_combo: "Combinaciones",

        quiz_title: "Quiz",
        quiz_question_fmt: "¿Cómo se dice \"%s\" en japonés?",
        review_label: "Repasa",

        item_rikai: "Rikai (gramática)",
        rikai_title: "Rikai",
        rikai_locked: "Aprende más vocabulario para desbloquear Rikai.",
        rikai_pick_help: "↑/↓ moverse · ENTER empezar · ESC volver",
        rikai_mastery_note: "Cada ronda cambia una sola palabra del patrón; el resto queda fija.",
        rikai_unlock_hint: "Aprende más palabras de este patrón primero.",
        rikai_question_fmt: "¿Cómo se dice \"%s\" aquí?",
        rikai_pattern_fluent: "dominado",
        rikai_mastered_fmt: "%d/%d",

        item_story: "Katsudoo (historia)",
        story_title: "Katsudoo",
        story_pick_help: "↑/↓ moverse · ENTER empezar · ESC volver",
        story_progress_fmt: "%d/%d escenas",
        story_complete_badge: "visto · reto pendiente",
        story_mastered_badge: "✓ dominado",
        story_empty: "Aún no hay capítulos disponibles.",
        story_done_title: "¡Capítulo dominado!",
        story_done_next: "ENTER volver a los capítulos",
        story_gate_note: "Cada capítulo se desbloquea dominando el anterior.",
        story_locked_hint_fmt: "Supera el reto de «%s» para desbloquear este capítulo.",
        story_present_label: "Aprende antes de practicar:",
        story_present_page_fmt: "Página %d de %d",
        story_present_more_help: "↑/↓ páginas · ENTER siguiente · ESC menú",

        story_challenge_title: "Reto del capítulo",
        story_challenge_intro_fmt: "Demuestra lo aprendido: acierta %d de %d para dominar el capítulo.",
        story_challenge_q_fmt: "Pregunta %d de %d",
        story_challenge_pass_fmt: "Reto superado: %d/%d.",
        story_challenge_fail_fmt: "Reto no superado: %d/%d (necesitas %d).",
        story_challenge_missed_lbl: "Para repasar:",
        story_challenge_retry_help: "ENTER reintentar · ESC salir",
        story_unlocked_fmt: "Desbloqueado: %s",

        item_assessment: "Examen N5",
        assessment_locked: "Domina todos los capítulos de la historia para desbloquear el examen.",
        assessment_passed_badge: "✓ aprobado",
        assessment_title: "Examen N5",
        assessment_intro_fmt: "Examen de nivel: acierta %d de %d para aprobar N5. Preguntas de vocabulario, kana y gramática.",
        assessment_best_fmt: "Mejor resultado: %d/%d.",
        assessment_pattern_prompt_fmt: "¿Qué palabra completa la frase? Pista: \"%s\".",
        assessment_pass_title: "¡N5 aprobado!",
        assessment_fail_title: "Examen N5",
        assessment_pass_fmt: "Aprobado: %d/%d.",
        assessment_fail_fmt: "No aprobado: %d/%d (necesitas %d).",
        assessment_missed_lbl: "Para repasar:",
        assessment_more_fmt: "+%d más",
        assessment_retry_help: "ENTER reintentar · ESC salir",
        assessment_done_help: "ENTER volver al menú",

        flash_title: "Flashcards",
        review_screen_title: "Repaso",
        reveal_help: "ESPACIO revelar · ESC menú",
        grade_prompt: "¿Qué tal lo recordaste?",
        grade_again: "Otra vez",
        grade_hard: "Difícil",
        grade_good: "Bien",
        grade_easy: "Fácil",
        reviewed_label: "Tarjetas repasadas",
        nothing_due: "No hay tarjetas para repasar ahora. Vuelve más tarde.",
        today: "hoy",
        day_short: "d",
        flash_new_held_fmt: "%d tarjetas nuevas en espera: entran poco a poco para consolidar lo aprendido.",
        freq_rank_fmt: "Frecuencia: nº %d",

        stats_title: "Mis estadísticas",
        best_label: "récord",
        hiragana_label: "Hiragana",
        katakana_label: "Katakana",

        welcome_title: "Bienvenido a Polyglot",
        welcome_intro: "Vas a aprender japonés desde el español.",
        controls_title: "Controles básicos:",
        welcome_next: "ENTER  probemos un ejercicio →",
        practice_title: "Práctica guiada",
        sample_word: "みず",
        sample_romaji: "mizu",
        sample_prompt: "¿Qué significa esta palabra?",
        sample_hint: "◀ pista: ¡es esta!",
        practice_correct: "¡Genial! Ya sabes lo básico.",
        practice_retry: "Casi… la respuesta correcta está marcada. Inténtalo.",
        practice_next: "ENTER  continuar →",
        done_title: "¡Todo listo!",
        done_recommend: "Te recomiendo empezar por el Entrenador de Kana.",
        done_next: "ENTER  ir al menú principal",
    }
    lists {
        controls_keys: [
            "↑ ↓      moverte por las opciones",
            "ENTER    confirmar",
            "ESPACIO  revelar respuesta (en flashcards)",
            "ESC      volver al menú",
            "Q        salir",
        ],
        sample_options: ["Fuego", "Agua", "Gato", "Árbol"],
    }
    ints {
        sample_correct: 1,
    }
}

/// The active UI language.
pub static DEFAULT: LazyLock<Messages> = LazyLock::new(Messages::es);

/// Returns the active (default) localization.
pub fn default() -> &'static Messages {
    &DEFAULT
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn tagline_uses_language_codes() {
        assert_eq!(Messages::es().tagline, "es → ja");
    }

    #[test]
    fn avoids_pictographic_emoji() {
        let banned = [
            "\u{1F1EA}\u{1F1F8}", // Spain flag
            "\u{1F1EF}\u{1F1F5}", // Japan flag
            "\u{1F3B4}",          // flower playing cards
            "\u{1F4CA}",          // bar chart
            "\u{1F525}",          // fire
            "\u{1F389}",          // party popper
            "\u{1F319}",          // crescent moon
            "\u{2728}",           // sparkles
            "\u{1F464}",          // bust silhouette
            "\u{267F}",           // wheelchair symbol
        ];
        let m = Messages::es();
        for value in m.all_strings() {
            for emoji in banned {
                assert!(
                    !value.contains(emoji),
                    "message {value:?} contains banned emoji {emoji:?}"
                );
            }
        }
    }

    #[test]
    fn uses_uppercase_key_labels() {
        let banned = ["enter", "esc", "espacio", "q"];
        let m = Messages::es();
        for value in m.all_strings() {
            for token in value.split(|c: char| !c.is_alphanumeric()) {
                assert!(
                    !banned.contains(&token),
                    "message {value:?} contains lowercase key label {token:?}"
                );
            }
        }
    }
}
