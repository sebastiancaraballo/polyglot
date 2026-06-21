// Package avatar generates small, text-only profile avatars. Avatars are derived
// deterministically from a name so they are stable, need no image rendering, and
// stay legible without color (each is a shape — initials or a block identicon — that
// callers may tint with the theme accent). This keeps the app accessible and
// dependency-free, unlike terminal image protocols.
package avatar
