# Scrape whole book as a png
- Requires the 4 character 'book' id from URL
- Example: `/books/xxxx/#p=1`, the id here is `xxxx`
- Hardcoded the URL of majalahsekolah, but you can modify it to use anyflip instead

## Strategy
- Download files in parallel, based on number of CPUs
- Convert every 25 file to separate PDFs
- Merge PDFs into a single PDF
- This prevents the program from exhausting all available memory
- If you have partial downloads, resuming is possible by invoking the same command again
    - Ensure the partially downloaded file is deleted beforehand
    - Something to do later: validate that pre-downloaded files are not corrupted

## Extension idea: Cloud based scraping
- Lambda based
- Each 'id' processed on a lambda
- Save result to bucket if exists (200 response)
- 'Master' lambda invokes with all existing IDs
- Should lambda do the downloading? Or just list valid IDs?
    - Download in lambda
        - need to store in bucket
        - lambda may timeout, takes like 5 mins to download all pages
        - maybe 1 lambda per page download
    - Save just valid IDs
        - need manual downloading
