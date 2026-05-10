# osamy

Misskey向けのリンクプレビューを返すやつ

### 特徴

- 特定サイトは専用スクレイパー、その他は汎用スクレイパーで処理する。
- デフォではRedisを使ったキャッシュを行い、失敗時はメモリにフォールバックする。

### エンドポイント

GET /?url={URL}

title, description, thumbnail, siteName, url, medias, player などを含むJSONを返す。

### 導入方法

docker-compose.ymlをサンプルファイルを見ながら編集する。  
CF Tunnnelで公開する際はHOSTは0.0.0.0のままでOKです。  
docker compose up -d --build を実行し、管理者設定で公開用に設定したURLをセットする。e.g.`https://example.com/`（スラッシュ必須）    
ローカル環境で検証を行う場合は`http://host.docker.internal:8080/`でビルドして実行する。
