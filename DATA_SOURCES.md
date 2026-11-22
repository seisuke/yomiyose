# テストデータの出典

ここには各データの入手元・抽出方法・利用条件と、規則の設計に参照した文献を記録する。どの名前をどこまでマスクできたかは [EVALUATION.md](EVALUATION.md)、ライブラリの使い方は [README.md](README.md) を参照。

| ファイル | 収録内容 | 名前の表記を得た方法 |
|---|---|---|
| [transcripts.tsv](testdata/transcripts.tsv) | 64文、必須のフルネーム71か所 | 利用例と手作りの文章テスト。一部に個人名典拠の異表記を使用 |
| [fleurs_names.tsv](testdata/fleurs_names.tsv) | 外国人11人、21出力（発話とモデルの組） | 日本語音声に対する正解文と実際のSTT出力から、フルネームの部分を抜粋 |
| [kotoba_kana_names.tsv](testdata/kotoba_kana_names.tsv) | 外国人10人、10発話 | Kotoba Technologies の Whisper large-v3 評価結果から、かなの正解名と実際の出力を抜粋 |
| [japanese_asr_names.tsv](testdata/japanese_asr_names.tsv) | 日本人7人、7発話（原表記の参考資料） | 日本語音声に対する正解文と実際のSTT出力から人物に対応する部分を抜粋。別列に2件のテスト用の読みを付与。公開STTのかな氏名のマスク性能集計からは全7件を除外 |

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

### 日本人名の資料をどこに反映したか

以下の距離は開発版の既定値による。発音研究から作った人名例と、公開STTで観測した出力は別に集計する。

| 資料・現象 | 実装・テストへの反映と未使用の情報 |
|---|---|
| 荒井（1999）の母音脱落と促音化 | 「聞くこと → きっこと」をもとに、日本人名向けの `KuGemination` を実装。手作りの「タナカキクコ → タナカキッコ」で距離 `0.5` とマスク範囲を検証した。「と思って」の連続母音に関しては、長音の脱落で表せる場合を既存規則で扱い、発話の持続時間は入力に使っていない。 |
| 前川の「キクチ」と Hara の姓名境界の母音 | キクチの無声化率から仮名「ク」が丸ごと消える率は求められず、「キクチユウスケ → キチユウスケ」は通常の語中脱落（`MedialLoss = 1`）としてテスト。Hara が測ったのは姓名境界の母音の持続時間。姓と名を分割せず読みを受け取る本ライブラリには、境界専用の母音脱落規則はない。 |
| CSJの長母音の短呼・文化庁の仮名遣い | 「サトウレーナ → サトウレナ」は `LongDeletion` で `0.75`、「サトウミホ → サトーミホ」は `OuSpelling` で `0`。CSJの語頭・語中・語末別の発生率を重みとして用いる処理はない。 |
| 浅井の姓末ハ／ワ・高橋・梅村の姓のかな異表記 | 「オトワユウスケ → オトはユウスケ」には `HaAsWa` を使う。「オトハユウスケ → オトワユウスケ」は通常の置換で距離 `1` で、姓に定着した異読を一律に置換する規則はない。語中の清濁は `MedialVoicing` で扱う。高橋・梅村の姓別の正規化規則群は取り込んでいない。[高橋・梅村の研究](https://ipsj.ixsq.nii.ac.jp/records/13848) |
| 日本人名のWhisper出力7件 | 「ケント → ケンタ」「ヒデヨ → ヒデオ」の2件は人手でかなを仮定し、末尾1文字を残す部分マスクを検証。残る5件は原表記を記録した。漢字の読みの推定と一般語への置き換わりは、かな照合の評価に含めていない。[7件の原表記と評価対象](testdata/japanese_asr_names.tsv) |
| CJKI の JEN・JNV、ELRA の人名音声 | JEN の姓名頻度による分類と ELRA 音声での評価は未実施。JNV の公開例には部分的な対応がある。「サトウミホ」と `satoumiho / satoomiho / satomiho` の距離は順に `0 / 0 / 0.75`。`satōmiho` は全体照合の対象外で、`satohmiho` は通常の置換で距離 `1`。両表記を長音として正規化する規則は未実装。撥音の後に母音が続く「ケンイチ」に対する `kenichi / ken’ichi` も正規化していない。[CJKI の公開例](https://www.cjk.org/language/ja/data/japanese/proper/database-japanese-personal-name-variants/) |

## 外国人名の発話別STT出力

### Kotoba Technologies の評価結果

JSUT basic5000 から3発話、Common Voice 8 日本語から2発話、ReazonSpeech test から5発話を選んだ。上記の固定コミットの各CSVで、`reference_raw` と `prediction_raw` に含まれる人物名の範囲を目視で対応付けた。`dataset` と `utterance` で元の行を特定できる。表記が一致する対照例、中黒の違い、ひらがなの混入や音の置換を含め、読みは補っていない。`expected_mask` の `=` は出力に名前がそのまま残ることを表す。

### FLEURS 日本語

音声と正解文の元データはGoogleのFLEURS日本語テストセット（CC BY 4.0）。認識結果は `yukobayashi500/ja-asr-leaderboard-2026` の `fleurs_outputs.jsonl` を使った。このファイルには読み上げ650発話に対する11モデル、計7,150行の出力がある。使用ファイルのSHA-256は `7c566d09af0ef0dfa3f36faea08e031ad974b452c8a8cd4bd2b2d6dbd4f7f46e`。[Google FLEURS](https://huggingface.co/datasets/google/fleurs)・[発話別結果の配布元](https://huggingface.co/datasets/yukobayashi500/ja-asr-leaderboard-2026)・[取得したJSONL](https://huggingface.co/datasets/yukobayashi500/ja-asr-leaderboard-2026/resolve/main/fleurs_outputs.jsonl)

比較できるフルネームがあり、既存の規則で拾える例と取りこぼす例を含む12行を選んだ。`utterance` は配布元が付けた `id`、`model` は同じ行にあるモデル識別子。各行の登録名は `reference`、観測名は `hypothesis` の部分文字列と照合して確認した。当初の12行はWhisper large-v3-turbo、large-v3、medium、Kotoba-Whisper v2.0の出力。後に、同じ10人を7,150出力の全文で照合し、長音コストの変更でマスク結果が変わった4行を追記した。いずれも同じJSONLにある別モデルの出力。周囲の文だけをテスト用に統一した。マスク期待値の `=` は観測名がそのまま残ることを示す。音声の聞き直し、独立の話者数の確認、この標本からの発生率の推定は行っていない。

### ラ行と D・T の聞き違い

大井川（2016）の知覚実験では、米語で母音間の /t/ がはじき音 [ɾ] になった音声を、日本語話者が主にラ行またはダ行に対応づけた。文化庁の日本語教育研究協議会には *better* の「ベター／ベラー」、*pudding* の「プリン」という例もある。国立国語研究所は、日本語でもラ行とダ行の混同を観察できると説明している。これらは音声上の根拠で、個別の日本語STT出力で何が発音されたかは特定できない。[大井川の知覚実験](https://www.isca-archive.org/isaph_2016/ooigawa16_isaph.html)・[文化庁の協議記録](https://www.bunka.go.jp/seisaku/kokugo_nihongo/kyoiku/taikai/16_kansai/bunkakai_2.html)・[国立国語研究所の解説](https://kotoba.ninjal.ac.jp/qa/yokuaru/qa-246/)

上記のFLEURSとKotobaの正解文・出力をNFKCでそろえ、中黒と空白を除いてモーラに近いかな単位で比較した。差分に現れた同母音のラ行とダ行・タ行の候補を原文で確認した。次の向きは正解表記からモデル出力への向きを表す。

| 正解の該当部分 → 実際の出力 | 発話とモデル | 観察 |
|---|---|---|
| ジャンカルロ・フィジケラ → ジャン・カルド・フィジケラ | fleurs-0602、Whisper large-v3-turbo | フルネームのロ → ド |
| カルパネド → カルパネロ | fleurs-0220、Qwen3-ASR-0.6B | 選手の姓のみ。ド → ロの逆方向 |
| リー・シェン・ルーン → ディー・シェン・ルーン | fleurs-0453、Whisper medium | フルネームの先頭でリ → ディ。SenseVoiceSmallも同じ発話で「ディーシェンルーン」と出力 |
| レアロフィ → デアロフィ | fleurs-0333、Whisper large-v3-turbo | フルネーム末尾のレ → デ。前半にも複数の違いがある |
| ディーン → リーン | ReazonSpeech `d5e36818c6db1.flac`、Whisper large-v3 | 「さん」の前の呼び名のみ。ディ → リの逆方向 |
| ラインダンス → ラインランス | ReazonSpeech `d027cb91bb410.flac`、Whisper large-v3 | 一般語のダ → ラ |
| レスラー → レスター | fleurs-0529、anime-whisper | 一般語のラ → タ |

`fleurs-0453` の anime-whisper 出力「ディ・シェン・ルーン」も、長音の落ちた語頭の回帰例として収録した。かなでフルネームが残る `fleurs-0162`（Whisper large-v3-turbo）と `fleurs-0603`（anime-whisper）も見落としを検証する行として収録した。TSV の `foreign_by_length_mask` は長さ別上限・指定閾値 `2` での期待マスク。この21件では標準設定と `ForeignRules` で同じ結果になり、両方を検証している。語中の「ロ」、末尾の「フィ」などを削った短縮例は別途Goテストで作成し、公開出力の件数から除外した。

フルネームがある `fleurs-0453` と `fleurs-0333` を評価用TSVに追記した。前者は語頭の照合、後者はラ行以外も変わる見落としの検証に使う。タ行については今回の抽出でタ行との置換だけで説明できるフルネームの例を確認できず、資料にとどめた。[発話別の評価結果](https://huggingface.co/datasets/yukobayashi500/ja-asr-leaderboard-2026)・[追加した表記対](testdata/fleurs_names.tsv)

### トゥプアの異表記と発音資料

サモアの独立運動指導者の綴りは **Tupua Tamasese Lealofi III**。FLEURS 日本語の `fleurs-0332`、`0333`、`0334` は正解文が同一の3発話である。Whisper large-v3-turbo の先頭は順に「トゥプア」「トワッパー」「トゥプア」。中央の発話では Whisper large-v3 が「トゥアッパー」、Whisper medium が「トワッパー」と出力した。同じ録音に対する複数モデルの表記なので、話者がどう発音したかや、別の人名での発生率はこの比較から決まらない。[ニュージーランド公文書館](https://www.archives.govt.nz/discover-our-stories/the-rise-of-the-mau-and-the-fall-of-colonial-rule-in-samoa)・[各モデルの出力](https://huggingface.co/datasets/yukobayashi500/ja-asr-leaderboard-2026)

- **トゥ／ト**。文化庁は同じ人名の表記例として「ハチャトゥリヤン／ハチャトリヤン」を挙げる。STTでも JSUT `BASIC5000_3315.wav` で「シャクロヴィートゥイ → シャクロビートイ」があり、ヴィ／ビとトゥ／トの2か所が対応する。[文化庁「外来語の表記」第6項](https://www.bunka.go.jp/kokugo_nihongo/sisaku/joho/joho/kijun/naikaku/gairai/honbun05.html)・[JSUTのWhisper large-v3出力](https://github.com/kotoba-tech/kotoba-whisper/blob/649fe3d1427d9ad6027940e29e7651fd516ea590/eval_pipeline/model-whisper-large-v3.dataset-ja_asr.jsut_basic5000.dataset_config-None.dataset_split-test.language-ja.task-transcribe.stable-ts-None.punctuator-None.chunk_length-15.csv)
- **プア／プワ**。Whisper large-v3 は `fleurs-0332` と `0334` で「トゥプワ」と出力した。Zuraw・Yu・Orfitelli はサモア語の一部の母音列で /u/ が [w] や [uw] として現れる例を報告する。一方、子音の直後では通常その転写をしなかったと明記している。Tupua の /p/ 直後への適用は実証されておらず、「プワ」と「ワッパー」ではワに対する子音の位置も異なる。[サモア語の音声研究、第8節](https://linguistics.ucla.edu/people/zuraw/dnldpprs/ZurawYuOrfitelli_Samoan_Phonology_2014.pdf)
- **ッパー**。英語からの借用では、同じ /p/ でも語や音環境によって促音化が異なる（cap の「キャップ」、captain の「キャプテン」）。英語の語末の -er 等をア列の長音で写す慣用もあるが、Tupua は -a で終わる。別の FLEURS 発話 `fleurs-0158` では Whisper large-v3 と turbo が人名以外の「チャッパン」を「チャップアン」と出力した。パ／プアが逆向きに現れた参考例で、この例では「ッ」は両方にある。[Kubozono・Itô・Mester の促音化研究](https://people.ucsc.edu/~ito/papers/2009_kubozono_ito_mester_gemination_CIL18.pdf)・[文化庁「外来語の表記」長音の項](https://www.bunka.go.jp/kokugo_nihongo/sisaku/joho/joho/kijun/naikaku/gairai/honbun06.html)

「トゥ → ト(ワ)」のうちワを伴う別の用例を探すため、FLEURS日本語の7,150出力と、上記のJSUT・ReazonSpeech・Common Voiceの計14,746出力を調べた。正解文と出力をNFKCで正規化し、ひらがなをカタカナにそろえて、正解文に「トゥ」、同じ行の出力に「トワ」を含む行を抽出した。該当したのは `fleurs-0333` の5モデルだけで、どれも同じ録音である。実装ではこの表記対を「トゥ → トワ」「プア → ッパ」として照合する。観測できるのは文字起こしの違いで、音がどう聞こえたかの解釈は仮説である。[FLEURSの発話別出力](https://huggingface.co/datasets/yukobayashi500/ja-asr-leaderboard-2026/resolve/main/fleurs_outputs.jsonl)

**「ウ」と「ワ」の音声学的な関係。** 米語の発音実験では、母音 [u] と半母音 [w] の区別に、舌や唇の動きのタイミングが強く関わった。標準日本語11話者のMRI研究では、「ウ」と「ワ」の子音で唇の開きに有意差はなく、「ウ」のほうが舌の奥と軟口蓋の間が狭かった。どちらも調音の研究で、「トゥ」「トワ」を聞き違える頻度は測っていない。[Burgdorf・Tilsen（2021）](https://doi.org/10.1016/j.wocn.2021.101073)・[前川（2020）第3.3節](https://icu.repo.nii.ac.jp/record/4799/files/06_Maekawa.pdf)

Tupua の最初の u は t と p の間にあり、サモア語の研究で [w] が典型的に観察された「次が母音」の位置から外れる。次が a なのは p の後の2番目の u である。日本語の文字起こしでも「トゥ」は1モーラ、「トワ」は2モーラで、後者には「ワ」の母音 a がある。この録音で最初の u が w に聞こえたかは、正解文とSTT出力だけでは特定できない。[Zuraw・Yu・Orfitelli（2014）第8節](https://linguistics.ucla.edu/people/zuraw/dnldpprs/ZurawYuOrfitelli_Samoan_Phonology_2014.pdf)

## 音の対応と重みを検討したライブラリ

Mondegreen（Apache-2.0）の音素対応表には、母音が同じときに比較できる子音 `sh / j` がある。yomiyose では FLEURS の `fleurs-0380` に直接現れた語中の「ジェ／シェ」を、既存のGoの置換規則に加えた。重みは yomiyose の `MedialVoicing` を使う。参照したコミットは `2cf793e2abdced743a8b6552f64ee7097a87b17c`。ソースコードや辞書の転載はない。[Mondegreenの音素対応表](https://github.com/NagaYu/mondegreen/blob/2cf793e2abdced743a8b6552f64ee7097a87b17c/mondegreen/phonetics.py#L221-L229)・[ライセンスと第三者データの記載](https://github.com/NagaYu/mondegreen/blob/2cf793e2abdced743a8b6552f64ee7097a87b17c/LICENSE)

kanasim（Apache-2.0）のコミット `f067a4e8be10d818b37b8b9f08de933dae01524b` にある `data/biphone` の音響距離表を、追加候補の比較実験に使用した。Goパッケージに表は同梱していない。たとえば「ビ」「ヴィ」はどちらも同じ子音・母音の組に割り当てられる。yomiyose ではこの差に調整可能な `BV` を使う。[kanasimのカナと音素の対応表](https://github.com/jiroshimaya/kanasim/blob/f067a4e8be10d818b37b8b9f08de933dae01524b/src/kanasim/data/biphone/kana2phonome_bi.csv)・[ライセンス](https://github.com/jiroshimaya/kanasim/blob/f067a4e8be10d818b37b8b9f08de933dae01524b/LICENSE)・[比較結果](docs/matching/candidates.md)

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
