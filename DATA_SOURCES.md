# テストデータの出典

ここには各データの入手元・抽出方法・利用条件と、規則の設計に参照した文献を記録する。どの名前をどこまでマスクできたかは [EVALUATION.md](EVALUATION.md)、ライブラリの使い方は [README.md](README.md) を参照。

| ファイル | 収録内容 | 名前の表記を得た方法 |
|---|---|---|
| [transcripts.tsv](testdata/transcripts.tsv) | 64文、必須のフルネーム71か所 | 利用例と手作りの文章テスト。一部に個人名典拠の異表記を使用 |
| [fleurs_names.tsv](testdata/fleurs_names.tsv) | 外国人10人、16出力（発話とモデルの組） | 日本語音声に対する正解文と実際のSTT出力から、フルネームの部分を抜粋 |
| [kotoba_kana_names.tsv](testdata/kotoba_kana_names.tsv) | 外国人10人、10発話 | Kotoba Technologies の Whisper large-v3 評価結果から、かなの正解名と実際の出力を抜粋 |
| [japanese_asr_names.tsv](testdata/japanese_asr_names.tsv) | 日本人7人、7発話 | 日本語音声に対する正解文と実際のSTT出力から人物に対応する部分を抜粋。別列に2件のテスト用の読みを付与 |

## テストデータの利用条件

Goモジュールのソース配布物には上記のTSVが含まれる。TSVを読むのはテストで、製品に組み込むGoパッケージは呼び出し側が渡したテキストだけを照合する。音声ファイルの同梱・取得は行っていない。[GoコードのMITライセンス](LICENSE)

- **FLEURS** — `fleurs_names.tsv` は Google の正解文と yukobayashi500 の評価結果から人名部分を抜き出し、マスク期待値を加えた。両配布元の CC BY 4.0 は商用利用も認めている。抜粋を再配布する際は、出典・ライセンス・変更内容の表示を引き継ぐ。[Google FLEURS](https://huggingface.co/datasets/google/fleurs)・[発話別の評価結果](https://huggingface.co/datasets/yukobayashi500/ja-asr-leaderboard-2026)・[CC BY 4.0](https://creativecommons.org/licenses/by/4.0/deed.ja)
- **Kotoba Technologies の評価結果** — JSUT、Common Voice 8、ReazonSpeech の各発話でどんな表記が観測されたかを、IDと人名部分の短い表記対で記録した。確認したコミットには評価CSV全体の商用再配布を明示するライセンスは見当たらない。文化庁は単なる事実・データやアイデアを参照して利用できると説明している。[参照した版](https://github.com/kotoba-tech/kotoba-whisper/tree/649fe3d1427d9ad6027940e29e7651fd516ea590)・[文化庁の説明](https://www.bunka.go.jp/seisaku/chosakuken/seidokaisetsu/chosakukensha_fumei/)
- **元の音声・台本** — JSUT は音声の営利利用について配布元への連絡を案内している。台本には出典別のライセンスがある。Mozilla は Common Voice の元データを CC0 と説明している。ReazonSpeech の配布元は日本語音声コーパスの用途を情報解析に限定している。これらは原資料を直接利用するときの条件である。[JSUTの利用条件](https://sites.google.com/site/shinnosuketakamichi/publication/jsut)・[JSUTの原著論文](https://arxiv.org/abs/1711.00354)・[Mozillaの解説](https://assets.mofoprod.net/network/documents/Creating-Community-Driven-Datasets-Report-032023-GIZ-Mozilla.pdf)・[ReazonSpeechの利用条件](https://research.reazon.jp/projects/ReazonSpeech/index.html)

## 日本人名の発話別STT出力

Kotoba Technologiesが公開した評価結果のうち、`openai/whisper-large-v3` のCSVを使用した。配布元のコミットは `649fe3d1427d9ad6027940e29e7651fd516ea590`。参照した行数はJSUT basic5000が5,000、ReazonSpeech testが5,263、Common Voice 8日本語が4,483。各CSVの `id` をテストデータの `utterance` に記録した。[JSUTの発話別CSV](https://github.com/kotoba-tech/kotoba-whisper/blob/649fe3d1427d9ad6027940e29e7651fd516ea590/eval_pipeline/model-whisper-large-v3.dataset-ja_asr.jsut_basic5000.dataset_config-None.dataset_split-test.language-ja.task-transcribe.stable-ts-None.punctuator-None.chunk_length-15.csv)・[ReazonSpeechの発話別CSV](https://github.com/kotoba-tech/kotoba-whisper/blob/649fe3d1427d9ad6027940e29e7651fd516ea590/eval_pipeline/model-whisper-large-v3.dataset-ja_asr.reazonspeech_test.dataset_config-None.dataset_split-test.language-ja.task-transcribe.stable-ts-None.punctuator-None.chunk_length-15.csv)・[Common Voice 8の発話別CSV](https://github.com/kotoba-tech/kotoba-whisper/blob/649fe3d1427d9ad6027940e29e7651fd516ea590/eval_pipeline/model-whisper-large-v3.dataset-ja_asr.common_voice_8_0.dataset_config-None.dataset_split-test.language-ja.task-transcribe.stable-ts-None.punctuator-None.chunk_length-15.csv)

`reference_raw` は配布元の正解文、`prediction_raw` は同じ発話のモデル出力に含まれる表記そのままの抜粋。たとえばReazonSpeechの `e52b25f09728b.flac` では正解文に「山崎賢人」、出力に「山崎健太」がある。Common Voiceの `common_voice_ja_20964452.mp3` は「長嶋茂雄 → 長崎市議」。姓名を確認でき、出力と比較できる箇所を目視で選んだ。発生割合を推定する無作為抽出ではない。

かな照合を試す2行のみ、`reference_reading` と `assumed_output_reading` を人手で付けた。たとえば「健太」に「ケンタ」という読みを仮定した。`expected_reading_mask` はその想定でかなの文を渡した結果。`-` は原表記だけを記録した行。「長崎市議」は人名が一般の語句に置き換わった出力のため、カタカナ人名の揺れの評価から除外した。[山﨑賢人の公式プロフィール](https://www.stardust.co.jp/talent/section2/yamazakikento/)

## 日本語の音が弱まる・落ちる条件を調べた音声資料

| 資料 | 録音で調べたこと | フルネーム照合との関係 |
|---|---|---|
| 荒井隆行（1999）、日本語母語話者の自然な電話独話 | 「聞くこと」/kikukoto/ → /kikkoto/、「書くこと」/kakukoto/ → /kakkoto/ を記録した。「と思って」では隣り合う二つの /o/ が一つの母音になり、持続時間が2モーラに満たない場合もあった。 | 前者は /u/ が落ちて後続の /k/ と促音を作る例。たとえば「タナカキクコ → タナカキッコ」は録音して検証できる仮説。論文が記録したのは一般の語句で、このフルネームは手作りの例。[ICPhS論文 §2.1.3–4](https://www.internationalphoneticassociation.org/icphs-proceedings/ICPhS1999/papers/p14_0615.pdf) |
| 前川喜久雄、日本語話し言葉コーパス（CSJ）の狭母音の無声化 | 姓「キクチ」/kikuci/ を、続けて無声化しうる /i/ と /u/ の例に挙げている。表6では無声子音に挟まれた /i/ の89.15%、/u/ の84.25%が無声化した。これは対象音全体の集計。 | 母音を発するときに声帯の振動が弱まったり消えたりしても、子音が残りうる。「キクチ」が「キチ」と転記される確率はこの数値から求められない。[CSJ分析、表6・図6](https://www2.ninjal.ac.jp/kikuo/KM_2009_ASBook.pdf) |
| Hara（2016）、日本人のフルネームの読み上げ | 姓末と名頭で異なる母音が続くフルネーム20通りを16人が各3回発声し、960発声のうち840件を分析した。連続する母音の平均持続時間は姓名の境界で約211 ms、別に測った実在語の内部で約270 msだった。 | 姓名を続けて言ったときに境目の母音がどう聞こえうるかを調べる資料。話速は統制しておらず、持続時間から文字の脱落率は推定できない。使用した姓名は付録2、計測値は表8。[Newcastle大学の博士論文](https://theses.ncl.ac.uk/jspui/handle/10443/3420) |
| 国立国語研究所、CSJの長母音の短呼 | 「本当」を「ホント」と短く発音する例を含む。短単位内の長母音の短呼率は語頭0.09%、語中3.77%、語末4.71%だった。 | 長音の位置を分けて調べる根拠になる。この集計には一般語が含まれ、入力が姓名連結なら姓末がどこかは分からない。[CSJ予備的分析 §6-4](https://clrd.ninjal.ac.jp/csj/misc/preliminary/j6_2.html) |

これらの資料の観測対象は人の発音。GPT Liveがカタカナで確定したフルネームの欠落例は今回確認できなかった。上記の件数はSTT回帰テストの分母に含めていない。`VowelLoss` の0.25は利用例「サイトウ／サトウ」用の設定値で、この発話研究から推定した重みではない。

### 日本人の姓にあるハとワ

姓「音羽」には「おとは」「おとわ」の両方が収録されている。浅井淳（2014）は、電話帳を使って「葉」「羽」などが姓末に来る場合の読みを集計し、山葉・稲葉・常葉の例を挙げている。「山葉寅楠」はヤマハの公式資料でも「やまはとらくす」と読まれている。これらは姓として定着した読みを示す資料。今回調べた範囲では、同じ人が姓から名まで続けて発音したとき、姓末のハがワまたはアとして転記された日本人フルネームの発話対は確認できなかった。[浅井（2014）§2.3–2.4](https://repository.ninjal.ac.jp/record/532/files/papers0702.pdf)・[日本姓氏語源辞典「音羽」](https://name-power.net/fn/音羽.html)・[ヤマハの創業者資料](https://archive.yamaha.com/ja/news_release/2019/19070801/)

テストの「オトワユウスケ → オトはユウスケ」は、登録したワに対応する位置にひらがなの「は」が出たと仮定した日本人名の例。Whisperで実測した「エルモア・ジェイムズ → エルモはジェームズ」とは別に検証している。

## 外国人名の発話別STT出力

### Kotoba Technologies の評価結果

JSUT basic5000 から3発話、Common Voice 8 日本語から2発話、ReazonSpeech test から5発話を選んだ。上記の固定コミットの各CSVで、`reference_raw` と `prediction_raw` に含まれる人物名の範囲を目視で対応付けた。`dataset` と `utterance` で元の行を特定できる。表記が一致する対照例、中黒の違い、ひらがなの混入や音の置換を含め、読みは補っていない。`expected_mask` の `=` は出力に名前がそのまま残ることを表す。

### FLEURS 日本語

音声と正解文の元データはGoogleのFLEURS日本語テストセット（CC BY 4.0）。認識結果は `yukobayashi500/ja-asr-leaderboard-2026` の `fleurs_outputs.jsonl` を使った。このファイルには読み上げ650発話に対する11モデル、計7,150行の出力がある。使用ファイルのSHA-256は `7c566d09af0ef0dfa3f36faea08e031ad974b452c8a8cd4bd2b2d6dbd4f7f46e`。[Google FLEURS](https://huggingface.co/datasets/google/fleurs)・[発話別結果の配布元](https://huggingface.co/datasets/yukobayashi500/ja-asr-leaderboard-2026)・[取得したJSONL](https://huggingface.co/datasets/yukobayashi500/ja-asr-leaderboard-2026/resolve/main/fleurs_outputs.jsonl)

比較できるフルネームがあり、既存の規則で拾える例と取りこぼす例を含む12行を選んだ。`utterance` は配布元が付けた `id`、`model` は同じ行にあるモデル識別子。各行の登録名は `reference`、観測名は `hypothesis` の部分文字列と照合して確認した。当初の12行はWhisper large-v3-turbo、large-v3、medium、Kotoba-Whisper v2.0の出力。後に、同じ10人を7,150出力の全文で照合し、長音コストの変更でマスク結果が変わった4行を追記した。いずれも同じJSONLにある別モデルの出力。周囲の文だけをテスト用に統一した。マスク期待値の `=` は観測名がそのまま残ることを示す。音声の聞き直し、独立の話者数の確認、この標本からの発生率の推定は行っていない。

## 音の対応と重みを検討したライブラリ

Mondegreen（Apache-2.0）の音素対応表には、母音が同じときに比較できる子音 `sh / j` がある。yomiyose では FLEURS の `fleurs-0380` に直接現れた語中の「ジェ／シェ」を、既存のGoの置換規則に加えた。重みは yomiyose の `MedialVoicing` を使う。参照したコミットは `2cf793e2abdced743a8b6552f64ee7097a87b17c`。ソースコードや辞書の転載はない。[Mondegreenの音素対応表](https://github.com/NagaYu/mondegreen/blob/2cf793e2abdced743a8b6552f64ee7097a87b17c/mondegreen/phonetics.py#L221-L229)・[ライセンスと第三者データの記載](https://github.com/NagaYu/mondegreen/blob/2cf793e2abdced743a8b6552f64ee7097a87b17c/LICENSE)

kanasim（Apache-2.0）のコミット `f067a4e8be10d818b37b8b9f08de933dae01524b` にある `data/biphone` の音響距離表を、追加候補の比較実験に使用した。Goパッケージに表は同梱していない。たとえば「ビ」「ヴィ」はどちらも同じ子音・母音の組に割り当てられる。yomiyose ではこの差に調整可能な `BV` を使う。[kanasimのカナと音素の対応表](https://github.com/jiroshimaya/kanasim/blob/f067a4e8be10d818b37b8b9f08de933dae01524b/src/kanasim/data/biphone/kana2phonome_bi.csv)・[ライセンス](https://github.com/jiroshimaya/kanasim/blob/f067a4e8be10d818b37b8b9f08de933dae01524b/LICENSE)・[比較結果](EVALUATION.md#kanasimで選んだ置換候補)

## 規則の参考資料

- 外国人名を同じ辞書見出しにまとめる際の差異と位置の分類。[国立国語研究所『現代日本語書き言葉均衡コーパス』形態論情報規程集 改定版 §2.5.3](https://repository.ninjal.ac.jp/record/2862/files/LR-CCG-08-03.pdf)
- 長音の仮名遣いと、ヴィ／ビなどの外来音の対応。[文化庁「現代仮名遣い」](https://www.bunka.go.jp/kokugo_nihongo/sisaku/joho/joho/kijun/naikaku/gendaikana/honbun_dai1.html)・[「外来語の表記」](https://www.bunka.go.jp/kokugo_nihongo/sisaku/joho/joho/kijun/naikaku/gairai/honbun05.html)
- 日本人の姓のかな表記を分析した近似照合の先行研究。[高橋・梅村「人名のかな表記のゆれに基づく近似文字列照合法」](https://cir.nii.ac.jp/crid/1050564287841082880)
- 同一人物の日本語表記を確認できる典拠。[Antonio Vivaldi](https://id.ndl.go.jp/auth/ndlna/00527045)・[David McKee](https://id.ndl.go.jp/auth/ndlna/00449504)

## 手作りの文章と名前の典拠

`transcripts.tsv` のイシイカオリ、ヤマモトユウスケ、サイトウ／サトウ、語尾の脱落などはライブラリの利用例を起点に作成した。`length-` の人工的な長音挿入や、同姓の別人を含む文、会議録、境界チェックも手作り。国籍やSTTモデルごとの発生頻度を示すデータではない。外国人名について以下の典拠から確認した差を組み込んだ。

本文、登録する向き、マスクすべき人物、期待するコストはテスト用に設定した。和語・外来語の仮名遣いと先行研究は規則を設計するための参考資料として参照した。[規則の参考資料](#規則の参考資料)

- デビ／デービとデビス／デービス。[国立国会図書館の David McKee](https://id.ndl.go.jp/auth/ndlna/00449504)・[国際日本文化研究センターの Jerome Dean Davis](https://sekiei.nichibun.ac.jp/GAI/ja/detail/?gid=GH080003&hid=3509)
- シェク／シェーク、ディケンズ／ディッケンズ、クリスティ／クリスティー。[Shakespeare](https://id.ndl.go.jp/auth/ndlna/00456207)・[Dickens](https://id.ndl.go.jp/auth/ndlna/00437850)・[Christie](https://id.ndl.go.jp/auth/ndlna/00435977)
- ヨハン／ヨーハン、ニナ／ニーナ、マリ／マリー。[Goethe](https://id.ndl.go.jp/auth/ndlna/00441109)・[Nina Fedoroff](https://id.ndl.go.jp/auth/ndlna/01169702)・[Marie Nicot](https://id.ndl.go.jp/auth/ndlna/033603867)
- エムバペ／エムバッペ。[Wikidataの日本語表記](https://www.wikidata.org/w/api.php?action=wbgetentities&ids=Q21621995&props=labels%7Caliases&languages=ja&format=jsonfm)
- 大小の仮名を含む異表記。Willis は名と姓の両方にあり、Morris は同じ規則が別の人物にも働く例。Wells は姓の表記差をフルネームの文で評価した。[William Willis](https://id.ndl.go.jp/auth/ndlna/00461075)・[William Morris](https://id.ndl.go.jp/auth/ndlna/00450584)・[Herbert George Wells](https://id.ndl.go.jp/auth/ndlna/00460562)
- トマス／トーマスの長音と、同一人物について名まで異なる例。典拠で姓のみ・頭文字付きの表記も、テスト文ではフルネームに組み込んだ。[Theodore Thomas](https://id.ndl.go.jp/auth/ndlna/00551703)・[Lowell Thomas](https://id.ndl.go.jp/auth/ndlna/00458627)
- スティーブン／スティーヴン。[Stephen King](https://id.ndl.go.jp/auth/ndlna/00445718)
- ヴィバルディ／ビバルディ。[Antonio Vivaldi](https://id.ndl.go.jp/auth/ndlna/00527045)

## GPT Liveと参考資料の扱い

本調査でGPT LiveはOpenAIの `gpt-live-transcribe` を指すものとした。日本語の実通話100件でこのモデルを比較した公開報告はあるが、今回確認した範囲では日本人フルネームごとの正解と確定出力を照合できる同モデルの公開データは見つからなかった。このリポジトリに収録した発話別STT結果は上記の他モデルによるもの。GPT Liveで検証する際は、本番と同じ言語・キーワード設定で、途中の字幕を表示する用途なら途中出力も、確定文は確定出力どうしで評価する。[OpenAIのリアルタイム文字起こし仕様](https://developers.openai.com/api/docs/guides/realtime-transcription)・[日本語の実通話での評価報告](https://zenn.dev/aj_1905/articles/stt-evaluation-method)

日本人名に絞って新しく測るための候補には、ELRAの `ELRA-S0228-56` がある。配布元によると日本人名2,000件を200人の話者が発声したコーパスで、利用条件はカタログにある。今回は音声もモデル出力も使用していない。[ELRAのコーパス仕様・利用条件](https://catalogue.elra.info/en-us/repository/browse/ELRA-S0228_56/)

## 読みの統計に使える資料

CJKIの日英人名データベース（JEN）は姓と名を別項目で持ち、かな読み、分類、ロケール、頻度を提供する。たとえば日本の姓「一井」の読みは「いちい」で、母音が続く位置が分かる。日本語で表記した中国名のサンプルもあるので、読みの傾向を集計するときはロケールと分類を使う。同社の日英人名データベースを使う場合、同じ読みを持つ漢字表記の数と実際の出現頻度は区別する必要がある。[CJKIの日英人名データベース](https://www.cjk.org/language/ja/data/japanese/proper/japanese-english-database-personal-names/)

利用案は、姓・名の読みから2〜3モーラの並びを集計し、登録されたフルネームの分割候補を内部で比較すること。姓と名の全組み合わせを用意する必要はない。外国人名のカナ読みも対照にして、長音を挿入してよい位置にどれだけ予測力があるかを別の人物で検証できる。STTの誤りやすさを推定するには、実際の文字起こしと正しい読みの対も必要になる。今回はCJKIの公開仕様と用例を確認した。全件データでの学習・評価は未実施で、モデルや統計表の配布条件は個別ライセンスの確認が必要になる。[CJKIのデータライセンス](https://www.cjk.org/language/ja/licensing/)

同社の日本人名異表記データベース（JNV）では、日本語名をローマ字にしたときに「とう」が `tō / tou / too / to / toh` と揺れる例を確認できる。これらはローマ字入力の評価に利用できる。[CJKIの日本人名異表記データベース](https://www.cjk.org/language/ja/data/japanese/proper/database-japanese-personal-name-variants/)
