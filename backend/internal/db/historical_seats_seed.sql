-- First install inserts the five székek. Existing rows (including admin edits) are left alone.
INSERT INTO historical_seats (name, name_ro, slug, content) VALUES
(
    'Csíkszék',
    'Ținutul Ciuc',
    'csikszek',
    $$## Csíkszék

A **Csíkszék** (románul *Ținutul Ciuc*) a székely székek egyike, történelmileg a **Csíki-medence** és környéke. Központja a hagyományos székely közigazgatásban **Csíkszereda** (Miercurea Ciuc) környéke volt.

Ma jórészt **Hargita megye** területére esik. A Csíkszékhez kötődő fiúszékek a néphagyományban a **Gyergyó-** és **Kászonszék**.

### Látnivalók, hagyomány

- Csíki-medence települései, templomok, népviselet, búcsúk
- Kapcsolódó megye: [Hargita megye](/hargita-megye)$$
),
(
    'Udvarhelyszék',
    'Ținutul Odorhei',
    'udvarhelyszek',
    $$## Udvarhelyszék

Az **Udvarhelyszék** (románul *Ținutul Odorhei*) a székely székek egyike; központja **Székelyudvarhely** (Odorheiu Secuiesc) környéke.

A történelmi székhez tartozó terület jelentős része ma is **Hargita megyéhez** tartozik.

### Jellegzetességek

- Székelyudvarhely város és környék települései
- Kapcsolódó megye: [Hargita megye](/hargita-megye)$$
),
(
    'Háromszék',
    'Trei Scaune',
    'haromszek',
    $$## Háromszék

A **Háromszék** (románul *Trei Scaune*, németül *Drei Stühle*) a legnagyobb kiterjedésű székely szék volt; központja **Sepsiszentgyörgy** (Sfântu Gheorghe) környéke.

Ma jórészt **Kovászna megye** területére esik (Sepsi-, Kézdi-, Orbai- és Miklósvárszék fiúszékekkel).

### Megye

- [Kovászna megye](/kovaszna-megye)$$
),
(
    'Marosszék',
    'Ținutul Mureș',
    'marosszek',
    $$## Marosszék

A **Marosszék** (románul *Ținutul Mureș*) központja **Marosvásárhely** (Târgu Mureș) környéke; a Maros völgye és a Mezőség kapcsolódó részei tartoztak ide.

A történelmi terület nagy része ma **Maros megyéhez** tartozik.

### Megye

- [Maros megye](/maros-megye)$$
),
(
    'Aranyosszék',
    'Ținutul Arieș',
    'aranyosszek',
    $$## Aranyosszék

Az **Aranyosszék** (románul *Ținutul Arieș*) a székely székek közül az **Aranyos völgyéhez** kötődő exklávé volt: a történelmi Magyarország nyugati részén, a **Fehér (Alba) megye** területén.

Ma Románia **Kolozs** és **Fehér** megyéinek határán felel meg; a székely hagyományok szempontjából a székelyföldi székekkel együtt szokás tárgyalni.

### Megjegyzés

Ez a szék nem esik egybe a mai három székelyföldi megyével (Hargita, Kovászna, Maros), de a Lámsza a teljes székely szék-hagyományt szeretné bemutatni.$$
)
ON CONFLICT (slug) DO NOTHING;
