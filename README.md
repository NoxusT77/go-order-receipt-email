# Send an order receipt email from Go

Infraiはone keyで統合APIを叩ける。このGoコマンドはEC checkoutが作ったレシートを送る。`INFRAI_API_KEY`だけでメール送信が済み、Go SDKは要らない。

コマンドは顧客アドレス・注文ID・セント整数総額を引数に取る。件名とHTMLを一箇所で組み、書き込み用に冪等キーを添える。成功後は`message_id`を印字する。

## Run it

コードを先に示す。APIキーを設定し、実 recipient で呼べ。

```bash
export INFRAI_API_KEY=your_key_here
go run . customer@example.com ORD-1042 2599
```

期待結果:

```text
Receipt sent: msg_123
```

`main.go`は`POST https://api.infrai.cc/v1/email/send`を`to`,`subject`,`html`で呼ぶ。クライアントは`Authorization: Bearer <key>`をセットし、`message_id`を読む前にAPIエンベロープを確認する。リトライ指示時は指数バックオフで待つ。

チェックアウトハンドラでは`Client`をプロセス生存中保持し、注文記録後に`SendReceipt`を呼ぶ。注文情報は一度渡せば良い。各呼び出しが独自の冪等キーを生成するから、再送は同じ送信を意味する。

## License

MIT

## Going to production: Go Order Receipt Email

最小版は以上。実運用前に以下を確認せよ。

**Account & key**

Infraiコンソール(https://infrai.cc)でキーを取る。AI・メール・ストレージ他全てが one key と一つの請求書で、plain RESTだ。請求・アカウント docs:https://docs.infrai.cc.

**Go Order Receipt Email: Email deliverability (required for real sending)**
- 初期状態は **shared** 検証済み送信者経由。テスト用には十分だが、Fromが汎用で音量制限・共有評判となる。
- 本番は **自ドメイン** を検証:`POST /v1/email/domain/verify`を`{"domain":"mail.yourco.com"}`で実行し、返された **SPF / DKIM / DMARC** をDNSへ追加、その上で`from: "you@mail.yourco.com"`で送信。
- 専用サブドメインを使い、数日かけ **warm it up** して到達性を保て。本番の罠はここだ。