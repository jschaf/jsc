+++
slug = "bitmaps-reveal-queen-bee"
date = 2025-03-21
visibility = "published"
+++

# Bitmaps reveal the Queen Bee

I fell into the Wordle craze a few years after everyone else. After indulging in
several months of Wordle, I shifted my focus to the NYT [Spelling Bee]. What
better way to enjoy the Queen Bee than to solve it programmatically? Any excuse
to play with bitmaps is fine by me.

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
a `uint32`. For example, to represent `citizen`:

```
0 0 1 0 1 0 0 0 1 0 0 0 0 1 0 0 0 0 0 1 0 0 0 0 0 1 0 0 0 0 0 0
a b c d e f g h i j k l m n o p q r s t u v w x y z _ _ _ _ _ _ 
```

When matching a word, we need to verify two conditions:

1. The center letter is included.
2. The word contains only the letters present in the `target` bitmap.

The challenge lies in ensuring that a word uses only the target letters. In
other words, does the `word` bitmap contain only the set bits corresponding to
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
	// center is a bitmap of the single center letter in the target word.
	center uint32
	// all is a bitmap of all letters in the target word
	all uint32
}

func newWorldMatcher(target string) wordMatcher {
	return wordMatcher{
		center: letterBitmap(target[0]),
		all:    wordBitmap([]byte(target)),
	}
}

func (m wordMatcher) isMatch(word []byte) bool {
	for _, r := range word {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	other := wordBitmap(word)
	hasCenter := other&m.center > 0
	isExclusive := (other & m.all) == other
	return hasCenter && isExclusive
}

func wordBitmap(word []byte) uint32 {
	bm := uint32(0)
	for _, ch := range word {
		bm |= letterBitmap(ch)
	}
	return bm
}

func letterBitmap(ch byte) uint32 { return 1 << (ch - 'a') }
```

## Results

The program works as advertised. For the target word `entivcz`, we find all
matching words in the dictionary. However, there are two problems with the
solver.

1. The list includes obscure words like `itcze` and `cetene`.
2. The list doesn't include suffixation, like adding `ed` or `ing` to the word.
   This means the solver misses the panagram for the puzzle: `incentivize`.

```go
Letters: etnizvc
Panagrams:
Matches:
  incentive
  inventive
  nineteen
  ...
  evict
  niece
  ...
  nine
  nice
```
