# osamy

Misskey向けのリンクプレビューを返すやつです。  
Misskeyには内臓のサマリープロキシがあるので、単体で動作可能ですが、  
負荷分散したい、独立して稼働させたいなどのユースケースでご活用いただけます。  

### 特徴

- 特定サイトは専用スクレイパー、その他は汎用スクレイパーで処理します。
  - 通常サイト、埋め込みコンテンツが伴う一部のサイト（Twitter,Bluesky,Threads,Spotify,Youtube etc）、オフィスドキュメント(PDF,Word,Excelファイル)をプレビュー表示可能です。 
- デフォではRedisを使ったキャッシュを行い、失敗時はメモリにフォールバックします。  
  - Redisのキャッシュ時間は24時間です。

### エンドポイント

GET /?url={URL} : プレビュー向けコンテンツに変換 (title, description, thumbnail, siteName, url, medias, player などを含むJSONを返す。)

GET /health : 動作確認用


### 導入方法

docker-compose.ymlをサンプルファイルを見ながら編集する。  
CF Tunnnelで公開する際はHOSTは0.0.0.0のままでOKです。  
docker compose up -d --build を実行し、管理者設定で公開用に設定したURLをセットする。e.g.`https://example.com/`（スラッシュ必須）    
ローカル環境で検証を行う場合は`http://host.docker.internal:8080/`でビルドして実行する。
