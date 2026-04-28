const CDN = "https://fonts.gstatic.com/s/e/notoemoji/latest"

const EMOJI = [
  { name: "grinning face",           keywords: ["happy", "smile", "joy", "grin"],          code: "1f600" },
  { name: "tears of joy",            keywords: ["laugh", "funny", "lol", "haha"],           code: "1f602" },
  { name: "rolling laughing",        keywords: ["rofl", "laugh", "funny", "floor"],         code: "1f923" },
  { name: "heart eyes",              keywords: ["love", "crush", "adore", "gorgeous"],      code: "1f60d" },
  { name: "sunglasses",              keywords: ["cool", "awesome", "shades"],               code: "1f60e" },
  { name: "thinking",                keywords: ["think", "hmm", "wonder", "ponder"],        code: "1f914" },
  { name: "mind blown",              keywords: ["wow", "surprised", "explode", "amazing"],  code: "1f92f" },
  { name: "screaming",               keywords: ["shock", "fear", "scared", "horror"],       code: "1f631" },
  { name: "loudly crying",           keywords: ["sad", "cry", "tears", "sob"],              code: "1f62d" },
  { name: "angry",                   keywords: ["mad", "rage", "furious", "pouting"],       code: "1f621" },
  { name: "eye roll",                keywords: ["sigh", "whatever", "seriously"],           code: "1f644" },
  { name: "sleeping",                keywords: ["sleep", "tired", "zzz", "night"],          code: "1f634" },
  { name: "partying",                keywords: ["party", "celebrate", "birthday", "fun"],   code: "1f973" },
  { name: "facepalm",                keywords: ["seriously", "ugh", "fail", "embarrass"],   code: "1f926" },
  { name: "shrug",                   keywords: ["idk", "whatever", "dunno", "unsure"],      code: "1f937" },
  { name: "thumbs up",               keywords: ["like", "good", "ok", "yes", "agree"],      code: "1f44d" },
  { name: "thumbs down",             keywords: ["dislike", "bad", "no", "disagree"],        code: "1f44e" },
  { name: "clapping",                keywords: ["applause", "bravo", "congrats", "well"],   code: "1f44f" },
  { name: "raising hands",           keywords: ["celebrate", "yes", "hooray", "praise"],    code: "1f64c" },
  { name: "flexed biceps",           keywords: ["strong", "muscle", "gym", "power"],        code: "1f4aa" },
  { name: "fire",                    keywords: ["hot", "flame", "lit", "burn"],             code: "1f525" },
  { name: "red heart",               keywords: ["love", "heart", "romance"],                code: "2764_fe0f" },
  { name: "broken heart",            keywords: ["heartbreak", "sad", "unlove"],             code: "1f494" },
  { name: "dizzy",                   keywords: ["stars", "sparkle", "spin", "dazed"],       code: "1f4ab" },
  { name: "collision",               keywords: ["boom", "explosion", "crash", "bang"],      code: "1f4a5" },
  { name: "zzz",                     keywords: ["sleep", "tired", "sleepy", "boring"],      code: "1f4a4" },
  { name: "rainbow",                 keywords: ["colorful", "pride", "color", "sky"],       code: "1f308" },
  { name: "party popper",            keywords: ["party", "celebrate", "confetti", "yay"],   code: "1f389" },
  { name: "birthday cake",           keywords: ["birthday", "cake", "candles", "bake"],     code: "1f382" },
  { name: "pizza",                   keywords: ["food", "dinner", "cheese", "italian"],     code: "1f355" },
  { name: "hamburger",               keywords: ["burger", "food", "lunch", "beef"],         code: "1f354" },
  { name: "beer",                    keywords: ["drink", "cheers", "pub", "ale"],           code: "1f37a" },
  { name: "coffee",                  keywords: ["morning", "cafe", "espresso", "hot"],      code: "2615" },
  { name: "cat",                     keywords: ["kitten", "meow", "feline", "pet"],         code: "1f431" },
  { name: "dog",                     keywords: ["puppy", "woof", "bark", "pet"],            code: "1f436" },
  { name: "fox",                     keywords: ["clever", "sly", "animal", "wild"],         code: "1f98a" },
  { name: "frog",                    keywords: ["kermit", "green", "ribbit", "pond"],       code: "1f438" },
  { name: "butterfly",               keywords: ["flutter", "colorful", "insect", "spring"], code: "1f98b" },
  { name: "cherry blossom",          keywords: ["flower", "spring", "pink", "japan"],       code: "1f338" },
  { name: "wave",                    keywords: ["ocean", "surf", "beach", "water", "sea"],  code: "1f30a" },
  { name: "lightning",               keywords: ["electric", "fast", "storm", "thunder"],    code: "26a1" },
  { name: "rocket",                  keywords: ["launch", "space", "fast", "startup"],      code: "1f680" },
  { name: "trophy",                  keywords: ["winner", "award", "first", "champion"],    code: "1f3c6" },
  { name: "video game",              keywords: ["game", "gaming", "controller", "play"],    code: "1f3ae" },
  { name: "ghost",                   keywords: ["halloween", "boo", "spooky", "scary"],     code: "1f47b" },
  { name: "unicorn",                 keywords: ["magic", "mythical", "horse", "fantasy"],   code: "1f984" },
  { name: "brain",                   keywords: ["think", "smart", "clever", "idea"],        code: "1f9e0" },
  { name: "musical note",            keywords: ["music", "song", "note", "sound"],          code: "1f3b5" },
  { name: "guitar",                  keywords: ["music", "rock", "instrument", "band"],     code: "1f3b8" },
  { name: "bullseye",                keywords: ["target", "aim", "goal", "hit", "darts"],   code: "1f3af" },
  { name: "globe",                   keywords: ["earth", "world", "global", "planet"],      code: "1f30d" },
]

function findEmoji(query) {
  if (!query) return []
  const q = query.toLowerCase()
  return EMOJI.filter(e =>
    e.name.includes(q) || e.keywords.some(k => k.includes(q))
  ).slice(0, 6)
}

app({
  state: {
    results: []
  },
  view: (state, actions) =>
    <main>
      <div class="results">
        {state.results.map(e =>
          <img
            key={e.code}
            src={`${CDN}/${e.code}/512.gif`}
            title={e.name}
          />
        )}
      </div>
      <input
        type="text"
        placeholder="Type here..."
        onkeyup={actions.search}
        autofocus
      />
    </main>,
  actions: {
    search: (state, actions, { target }) => ({
      results: findEmoji(target.value)
    })
  }
})
