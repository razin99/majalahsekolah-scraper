# Scrape whole book as a png
- Requires the 4 character 'book' id from URL
- Example: `/books/xxxx/#p=1`, the id here is `xxxx`

## Strategy
### Strategy A
- Download in chunks of 50
- If any response in chunk does not reply with a 200, stop
### Strategy B
- Spawn thread while all response returns 200
- Thread that has a non-200 response will stop the loop
