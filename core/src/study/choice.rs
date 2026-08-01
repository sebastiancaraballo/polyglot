use std::collections::HashSet;

use rand::seq::SliceRandom;
use rand::Rng;

/// Returns up to `n` answer options containing `correct` plus distinct
/// distractors drawn from `pool`, along with the index of the correct option.
/// The ordering is randomized using `rng` so the result is deterministic in
/// tests.
pub fn options<R: Rng + ?Sized>(
    rng: &mut R,
    correct: &str,
    pool: &[String],
    n: usize,
) -> (Vec<String>, usize) {
    let mut seen: HashSet<&str> = HashSet::new();
    seen.insert(correct);
    let mut distractors: Vec<String> = Vec::with_capacity(pool.len());
    for p in pool {
        if seen.insert(p.as_str()) {
            distractors.push(p.clone());
        }
    }
    distractors.shuffle(rng);

    let want = n.saturating_sub(1).min(distractors.len());
    let mut opts = Vec::with_capacity(want + 1);
    opts.push(correct.to_string());
    opts.extend_from_slice(&distractors[..want]);
    opts.shuffle(rng);

    let correct_idx = opts.iter().position(|o| o == correct).unwrap_or(0);
    (opts, correct_idx)
}

#[cfg(test)]
mod tests {
    use super::*;
    use rand::rngs::StdRng;
    use rand::SeedableRng;

    fn pool(items: &[&str]) -> Vec<String> {
        items.iter().map(|s| s.to_string()).collect()
    }

    #[test]
    fn includes_correct_and_indexes_it() {
        let mut rng = StdRng::seed_from_u64(1);
        let (opts, idx) = options(&mut rng, "a", &pool(&["b", "c", "d", "e"]), 4);
        assert_eq!(opts.len(), 4);
        assert_eq!(opts[idx], "a");
        assert!(opts.contains(&"a".to_string()));
    }

    #[test]
    fn dedups_pool_and_caps_gracefully() {
        let mut rng = StdRng::seed_from_u64(2);
        // Pool has duplicates and the correct answer; only 2 distinct distractors.
        let (opts, idx) = options(&mut rng, "a", &pool(&["a", "b", "b", "c"]), 4);
        assert_eq!(opts.len(), 3); // correct + 2 distinct distractors
        assert_eq!(opts[idx], "a");
    }
}
