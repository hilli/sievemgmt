require ["imap4flags"];

# Example IMAPSIEVE script: keep messages in the associated mailbox unread.
# Associate this script with a mailbox using:
#
#   sievemgmt upload examples/mark-unseen-imapsieve.sieve mark-unseen
#   sievemgmt folders set TODO mark-unseen

removeflag "\\Seen";
keep;
