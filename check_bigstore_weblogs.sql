-- Dremel/Plex SQL script to check GCSFuse User-Agent modifications in bigstore_web_logs.
-- Replace '<bucket-name>' with your specific bucket name if needed.

SELECT
  timestamp,
  operation,
  bucket_name,
  user_agent
FROM
  `bigstore_web_logs` -- Or the appropriate colossus.bigstore.web_logs dataset
WHERE
  user_agent LIKE '%Hello Pranjal is making changes in the useragent!%'
  AND bucket_name = '<bucket-name>'
ORDER BY
  timestamp DESC
LIMIT 100;
