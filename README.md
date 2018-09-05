# GoLang-FileBrowser
## App Structure
Using Golang, the application listens for an HTTP request to be sent on port 8080. Once a valid request is received, the user is able to navigate throughout the server's file directory and download/view the files within.

## Tests
### Directory Paging
- Create a large number of files in a directory then navigate to that directory. Check to see if the paging list has no more than the paging limit and test if the cursory query is functional
### Test for properly handled errors
- Attempt to load a series of invalid pages to see if any result in an uncaught error (e.g., server not responding, server-side fatal logs).

## Missing Requirements
- Should the sorted directory results show directory contents in an order that disregards the file type?
- Should partial file downloading be handled where only a portion of the file is being previewed and not being permanently downloaded (Similar to how you can preview books through services such as Google Books or Amazon Kindle)? If not, what does an alternative to this system look like?
- How should this application handle errors (e.g., HTTP errors, go fatals)?
- Should paging be handled by queries?