# File based database

 See API and examples here https://godoc.org/github.com/reddec/filedb

 Main features:

 ## 1. File based and human readable

 All items are saved as single JSON (may be changed in future) file.
Tables (in terms of RDB) is just sub-folder. Names encoded by URL encoding.
So, any administrator can fix and update item just using favorite text editor.

## 2. Events and reactive design

You can manipulate with items outside application any time. Application may receive notification about any changes in sections. 

## 3. Simple

Really simple structure of database engine in really simple language - Go. So pull requests are welcome
