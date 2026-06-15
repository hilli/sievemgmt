require [ "fileinto", "envelope", "reject", "mime", "imap4flags" ];

if address :is "to" [
  "me@example.com",
  "alsome@example.com"
] {
  fileinto "INBOX";
}

if address :is "from" [
  "microsoftfamily@microsoft.com",
  "mailserver-report@mail.shd.dk"
  ] {
  fileinto "Rapporter";
}

if header :contains [ "X-Spam-Flag"] [ "YES" ]
{
  fileinto "SPAM";
  stop;
}

if allof (
        address :matches "From" ["*@gmail.com"],
        not header :matches "Subject" "*[TEKNIK]*"
        )
{ redirect "me@gmail.com"; keep; }
