+++
slug = "bitmaps-reveal-queen-bee"
date = 2025-03-21
visibility = "published"
+++

# Bitmaps reveal the Queen Bee

I fell into the Wordle craze a few years after everyone else. After indulging in
Wordle for several months , I shifted my focus to the NYT [Spelling Bee]. What
better way to enjoy the Spelling Bee than solving it programmatically? Any
excuse to play with bitmaps is fine by me.

:embed: {name='bitmaps-reveal-queen-bee.html'}

[Spelling Bee]: https://www.nytimes.com/puzzles/spelling-bee

The rules of the NYT Spelling Bee are:

1. Words must contain at least four letters.
2. Words must include the center letter.
3. The word list excludes obscure words, proper nouns, and hyphenated words.
4. No naughty words.
5. Letters can be reused.

## Sketch

I started with the following sketch:

- Build a Go command line app, invoked as `spelling-bee --letters=entivcz`.
- The first letter is the center letter.
- Loop through `/usr/share/dict/words` to find matching words.
- Display panagrams first, followed by other matching words sorted by descending
  length.

To match the words, we'll use a bitmap representing the character set of a word.
Since there are 26 letters, and we don't care about frequency, the data fits in
a 32-bit unsigned integer. For example, to represent `citizen`:

```
  1 1   1    1     1     1      
abcdefghijklmnopqrstuvwxyz______
```

When matching a word, we need to verify two conditions:

1. The center letter is included.
2. The word contains only the letters present in the `target` bitmap.

The challenge lies in ensuring that a word uses only the target letters. In
other words, does the `word` bitmap contain only the set bits that correspond to
the `target` bitmap? We can encode the logic with two steps:

1. Bitwise `and` the `word` bitmap with the `target` bitmap. The result is only
   the letters present in both bitmaps.
2. Check that the `word` bitmap is the same as step 1.

```
(word & target) == word

# Equivalently
(word & target) ^ word == 0 
```

## Go program

See full code at the GitHub [Gist]. The main logic is in the `wordMatcher`.
We build the matcher from the target letters and match words from the
dictionary.

[Gist]: https://gist.github.com/jschaf/c7e282cb092df2e5ae80254d13e3b8a3

```go {description="match words with bitmaps"}
package main

type wordMatcher struct {
	// bitmap of the center target letter
	center uint32
	// bitmap of all target letters
	all uint32
}

func newWordMatcher(target string) wordMatcher {
	return wordMatcher{
		center: letterBitmap(target[0]),
		all:    wordBitmap([]byte(target)),
	}
}

func (m wordMatcher) isMatch(word []byte) bool {
	// Skip proper nouns and hyphenated words.
	for _, r := range word {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	other := wordBitmap(word)
	hasCenter := other&m.center > 0
	isTargetLetters := (other & m.all) == other
	return hasCenter && isTargetLetters
}

func wordBitmap(word []byte) uint32 {
	bm := uint32(0)
	for _, ch := range word {
		bm |= letterBitmap(ch)
	}
	return bm
}

func letterBitmap(ch byte) uint32 {
	return 1 << (ch - 'a')
}
```

## Results

The program works as advertised. For the target letters `entivcz`, we find all
matching words in the dictionary. Using the word list assembled by Reddit user
[ahecht], we find the panagram `incentivize` and other matches like `incentive`.
The dictionary may be incomplete, so a fun next project would be extracting the
word list from the app.

[ahecht]: https://old.reddit.com/r/NYTSpellingBee/comments/zzdo3q/rebuilding_spelling_bee_for_fun_what_dictionary/

```txt
Letters: entivcz
Panagrams:
  incentivize
Matches:
  incentive
  ...
  nineteen
  evictee
  ...
  vine
  zine
```
