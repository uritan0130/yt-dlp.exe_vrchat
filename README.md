# VRChat yt-dlp Wrapper

VRChatでYouTube動画を再生した際に、**「映像は出るのに音が出ない」**、または一部の動画URLがAVProで開けない症状が自環境で発生したため作成した、**非公式のyt-dlpラッパー**です。

> [!WARNING]
> - VRChat Inc. 公式のツールではありません。
> - yt-dlp / Deno / YouTube / Google の公式ツールでもありません。
> - VRChatやYouTube側の仕様変更で突然動かなくなる可能性があります。
> - 使用前にVRChatの最新の利用規約等を確認し、使用は各自の判断でお願いします。
> - DRM回避を目的としたツールではありません。

## 症状と原因

今回の環境では、通常のyt-dlpで `itag=137` / `itag=399` などの**映像専用ストリーム**が選択され、VRChatでは映像だけ再生されて音声が出ないケースがありました。

また、映像＋音声が1本に入ったURLを取得できても、Googlevideo側のURLが一時的に利用できず、AVProで `Loading failed` になる場合がありました。

このラッパーでは、主に次の処理を行います。

- DenoをJavaScript runtimeとして利用
- YouTubeの `web_embedded` クライアントを優先
- 映像＋音声を含む単一ストリームだけを選択
- H.264/AAC HLS → 音声付きHLS → 音声付きMP4 → その他の単一A/V の順に選択
- `--check-formats` で形式を確認
- VRChatへURLを返す前にHTTP Range requestでアクセス可能か検査
- 失敗時は再取得・別クライアントへフォールバック

現在のYouTubeでは `itag=18`（360p MP4・映像＋音声）が選択されることが多いため、**安定性優先で画質が360pになる場合があります。**

## 使い方

1. **VRChatを起動します。**
2. HomeまたはWorldに入り、VRChat側のURL Resolver更新が終わるまで待ちます。
3. このリポジトリの `yt-dlp.exe` をダウンロードします。
4. 次のフォルダを開きます。

```text
%USERPROFILE%\AppData\LocalLow\VRChat\VRChat\Tools
```

5. 元の `yt-dlp.exe` を必要ならバックアップします。
6. ダウンロードした `yt-dlp.exe` に置き換えます。
7. **VRChatは再起動せず**、YouTube動画を再生して確認します。

> VRChatを再起動すると、VRChat側のURL Resolverに戻される場合があります。その場合は、VRChat起動後にもう一度差し替えてください。

## 元に戻す方法

1. `Tools` フォルダ内のこの `yt-dlp.exe` を削除します。
2. VRChatを再起動します。

VRChat側のURL Resolverが再取得されます。

このツールが作成したキャッシュも削除したい場合は、次を削除してください。

```text
%LOCALAPPDATA%\VRChatYTDLP-OneFile\
```

## One-File版について

VRChatの `Tools` フォルダに置くファイルは `yt-dlp.exe` 1個だけです。

内部では必要に応じて次の固定バージョンを公式GitHub Releasesから取得し、ローカルキャッシュへ保存します。

- yt-dlp: `2026.08.19`
- Deno: `2.9.5`

ダウンロードした依存ファイルはハッシュ検証を行います。

## ログ

ラッパー自身のログは次に保存されます。

```text
%LOCALAPPDATA%\VRChatYTDLP-OneFile\yt-dlp-onefile.log
```

VRChatの `output_log` にはIPアドレスや期限付き・署名付きGooglevideo URLなどが含まれる場合があります。Issue等へ貼る場合は内容を確認してください。

## ネットワークアクセス

このツールは機能上、次の通信を行います。

- YouTube / Googlevideo: 動画URLの解決・事前検査
- GitHub Releases: 初回のyt-dlp / Deno取得

独自のテレメトリ送信機能は実装していません。

## ソースコード

`src/main.go` に公開しています。

Go標準ライブラリのみでビルドできます。詳しくは `BUILD.md` を参照してください。

## ライセンス

- このラッパー: MIT License (`LICENSE`)
- yt-dlp: Public Domain / Unlicense 相当
- Deno: MIT License

詳細は `THIRD_PARTY_NOTICES.md` を参照してください。

## Disclaimer

This project is not affiliated with, endorsed by, or sponsored by VRChat Inc., yt-dlp, the Deno project, Google, or YouTube.
