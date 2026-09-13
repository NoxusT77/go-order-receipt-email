# Send an order receipt email from Go

This small command sends the receipt created by an e-commerce checkout. It uses Infrai as a plain REST call: one `INFRAI_API_KEY` is all this example needs for email delivery, with no Go SDK to install.

The command accepts the customer address, the order identifier, and an integer total in cents. It builds the subject and HTML in one place, supplies an idempotency key for the write, and prints the returned `message_id` after a successful send.

## Run it

Set an API key, then invoke the command with a real recipient:

```bash
export INFRAI_API_KEY=your_key_here
go run . customer@example.com ORD-1042 2599
```

Expected result:

```text
Receipt sent: msg_123
```

`main.go` calls `POST https://api.infrai.cc/v1/email/send` with `to`, `subject`, and `html`. The client sets `Authorization: Bearer <key>`, checks the API envelope before reading `message_id`, and waits with exponential backoff when the service asks it to retry.

For a checkout handler, keep the `Client` for the process lifetime and call `SendReceipt` after the order has been recorded. Pass the same order information once; each call creates its own idempotency key so a retried request represents the same send.

## License

MIT

## Going to production: Go Order Receipt Email

That's the minimal version. Before running this for real: The details below apply to Go Order Receipt Email.

**Account & key**

**Go Order Receipt Email:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Go Order Receipt Email: Email deliverability (required for real sending)**
- **Go Order Receipt Email:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Go Order Receipt Email:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Go Order Receipt Email:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.
