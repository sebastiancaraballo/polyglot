package art

// Wordmark is the block-letter app name shown in the main menu header. It echoes
// the larger "ANSI Shadow" logo in the README, drawn full-block so it reads the
// same, but sized to span the shared fixed-width frame (4 rows, 55 columns). The
// README's six-row 3D logo is 66 columns wide and cannot fit a 64-column frame
// beside the menu, so this is a width-filling rendering of the same name in the
// same solid-block style. It uses only the full block glyph (█) and spaces, so
// it renders consistently across terminals and fonts.
const Wordmark = "██████ ██████ ██     ██  ██ ██████ ██     ██████ ██████\n" +
	"██  ██ ██  ██ ██      ████  ██     ██     ██  ██   ██  \n" +
	"██████ ██  ██ ██       ██   ██ ███ ██     ██  ██   ██  \n" +
	"██     ██████ ██████   ██   ██████ ██████ ██████   ██  "
