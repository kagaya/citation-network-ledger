# Citation Network Ledger Go 3.0.2

**作成者:** Katsushi Kagaya  
**連絡先:** kkagaya@excyberlab.com  
**著作権:** Copyright (c) 2026 Katsushi Kagaya  
**コードライセンス:** MIT License

PDFを一つずつ追加しながら引用ネットワークを育てる、ブラウザー不要のCUI台帳です。研究分野には依存しません。

## 公開版に含まれるデータ

同梱のザリガニ神経行動学サンプル台帳には、書誌情報、参考文献、引用辺、照合状態を含めています。論文PDF、Google Driveリンク、ローカルPDFパスは含めていません。

## 特徴

- 台帳操作は単一の実行ファイルだけで動作
- 追加の言語ランタイムやパッケージ管理ツールのインストールは不要
- Windows、WSL/Linux、Intel Mac、Apple Silicon Mac用バイナリ
- 台帳は可読なUTF-8 `ledger.json` として原子的に更新
- PDFはプロジェクト内の `pdfs/` にコピー
- CSV、JSON、Graphviz DOT形式で書き出し
- ザリガニ神経行動学の既存台帳をサンプルとして実行ファイルに内蔵

## 実行ファイルの選択

| 環境 | 実行ファイル |
|---|---|
| Windows 64-bit | `citation-ledger-go-3.0.2-windows-amd64.exe` |
| WSL / Linux 64-bit | `citation-ledger-go-3.0.2-linux-amd64` |
| Intel Mac | `citation-ledger-go-3.0.2-darwin-amd64` |
| Apple Silicon Mac | `citation-ledger-go-3.0.2-darwin-arm64` |

以下では実行ファイルを `citation-ledger-go` に改名したものとして説明します。

## WSL/Linuxで使う

```sh
chmod +x citation-ledger-go
./citation-ledger-go doctor
```

空のプロジェクトを作成します。

```sh
./citation-ledger-go --data-dir ~/citation_projects/limpet \
  init --name "Limpet Homing"
```

同梱されているザリガニ文献台帳から開始する場合は次のようにします。

```sh
./citation-ledger-go --data-dir ~/citation_projects/crayfish \
  init --seed-crayfish
```

`--data-dir`を省略すると、現在のディレクトリに `citation-ledger-data/` を作ります。環境変数 `CITATION_LEDGER_DATA_DIR`でも指定できます。

## Windowsで使う

PowerShellまたはコマンドプロンプトで実行できます。WSLは不要です。

```powershell
.\citation-ledger-go-3.0.2-windows-amd64.exe doctor
.\citation-ledger-go-3.0.2-windows-amd64.exe --data-dir C:\citation_projects\crayfish init --seed-crayfish
.\citation-ledger-go-3.0.2-windows-amd64.exe --data-dir C:\citation_projects\crayfish status
```

この配布物には商用コード署名を付けていないため、Windowsが初回実行時に警告することがあります。`SHA256SUMS.txt`の値と照合してから実行してください。WSLではLinux版を使うため、この警告はありません。

## 基本コマンド

```sh
# 状態
./citation-ledger-go --data-dir PROJECT status

# 文献一覧と検索
./citation-ledger-go --data-dir PROJECT papers
./citation-ledger-go --data-dir PROJECT papers --search "readiness"

# 文献の詳細と前後の引用
./citation-ledger-go --data-dir PROJECT show kagaya_takahata_2010

# 未解決参考文献と収集優先順位
./citation-ledger-go --data-dir PROJECT unresolved -n 30
./citation-ledger-go --data-dir PROJECT queue -n 30

# 引用辺
./citation-ledger-go --data-dir PROJECT edges
./citation-ledger-go --data-dir PROJECT edges --dot > network.dot

# JSON出力
./citation-ledger-go --json --data-dir PROJECT status
```

## PDFと参考文献の登録

確認・修正済み参考文献テキストがある場合は、外部プログラムを一切使いません。

```sh
./citation-ledger-go --data-dir PROJECT add paper.pdf \
  --title "Paper title" \
  --authors "Author, A.; Collaborator, B." \
  --year 2026 \
  --venue "Journal name" \
  --doi "10.xxxx/example" \
  --tags "behavior;neuroethology" \
  --references references.txt
```

`--references`を省略すると、`pdftotext -layout`でPDF本文を抽出します。WSL/Ubuntuでは必要に応じて次を実行します。

```sh
sudo apt install poppler-utils
```

PDFから参考文献節だけを確認用ファイルへ出すこともできます。

```sh
./citation-ledger-go extract paper.pdf -o references.txt
```

画像だけのスキャンPDFにはOCRが必要です。OCRmyPDFまたはTesseractで文字層を付けてから登録してください。OCRは自動実行しません。

## 引用の確認

照合待ちIDを既存文献へ結びます。

```sh
./citation-ledger-go --data-dir PROJECT resolve 123 wine_krasne_1972
```

未収集文献を候補ノードにします。

```sh
./citation-ledger-go --data-dir PROJECT candidate 123 \
  --title "Candidate paper" --authors "Author" --year 1974
```

DOI完全一致、著者・年・表題語の一致から登録時に自動照合も行います。自動確定されなかった項目は `unresolved` で確認できます。

## 書き出しとバックアップ

```sh
./citation-ledger-go --data-dir PROJECT export -o ledger_backup.zip
./citation-ledger-go --data-dir PROJECT export -o ledger_with_pdfs.zip --include-pdfs
```

ZIPには `ledger.json`、`papers.csv`、`raw_references.csv`、`citations.csv`、`settings.csv` が入ります。通常の書き出しにはPDFを含めません。

```sh
./citation-ledger-go --data-dir PROJECT validate
```

## データ形式

```text
PROJECT/
├── ledger.json
└── pdfs/
    └── paper_id_original_name.pdf
```

`ledger.json`には文献、参考文献原文、照合状態、引用辺が入ります。保存時には一時ファイルを書き終えてから置き換えるため、途中終了で半端なJSONを残しにくい設計です。同じプロジェクトを複数端末から同時に編集することは避けてください。

## 依存関係

台帳操作、検索、照合、書き出しには追加依存がありません。

| プログラム | 必須性 | 用途 |
|---|---|---|
| `pdftotext`（Poppler） | 任意 | 文字PDFの本文抽出 |
| `qpdf` | 任意 | 壊れたPDFの修復・正規化 |
| OCRmyPDF / Tesseract | 任意 | スキャンPDFのOCR |
| Graphviz | 任意 | DOTファイルの画像化 |

## ソースからビルド

利用者はGoを導入する必要はありません。ソースを変更して再ビルドする場合だけGo 1.22以上を使用します。

```sh
sh build_all.sh
```

Goの外部モジュールは使っていません。標準ライブラリだけでビルドします。

## 作成者・著作権・ライセンス

- 作成者: Katsushi Kagaya
- 連絡先: kkagaya@excyberlab.com
- 著作権: Copyright (c) 2026 Katsushi Kagaya
- コード・実行ファイル: MIT License（`LICENSE.txt`）
- 先生が作成した説明・引用辺・注釈: CC BY 4.0（`DATA_LICENSE.txt`）
- 出版社PDF・第三者由来の文献内容: 同梱せず、本配布物からはライセンスしません。

ソースコードは研究成果の確認可能性と将来の保守のために同梱しています。MIT Licenseの条件に従い、利用・改変・再配布できます。
