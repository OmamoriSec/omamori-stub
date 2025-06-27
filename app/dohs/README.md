### For sending DNS queries and getting DNS responses over HTTPS.

#### Request format:
```bash
GET: /dns-query?dns=BASE64URLENCODED_DNS_QUERY
HOST: omamori.com
Accept: application/dns-message

# dns is the encoded binary dns query
```

```bash
POST /dns-query
Host: omamori.com
Content-Type: application/dns-message
Accept: application/dns-message
Content-Length: 33

<raw binary DNS query>

```

#### Response Format
The response is always in binary DNS wire format `application/dns-message`
Same format as DNS response over udp