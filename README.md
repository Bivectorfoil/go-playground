### go-playground

> Attention: Just for fun. DO NOT USE IN PRODUCTION.

This project is a simple playground for Go language, it's a web page that allows you to write, run and see the output of Go code.

I created it just for fun, and to see how easy to build a simple project with [Cursor](https://www.cursor.com/). It takes me like 2~3 hours (more or less, can't remember exactly) to build this project, all i do is to chat with Cursor, create directory, file as needed, and click `apply` from Cursor to apply the changes, also to test the UI/UX of frontend, ask Cursor to fix issues i meet or adjust the functionality that has been built.

It's a pity that i can't export the chat history with you guys, Cursor hasn't supported this feature yet. It's takes me about 20~30 rounds (maybe more, can't remember exactly) of iteration to get the project done. The code quality is not good, but the project works well, so don't take it seriously.

### Prerequisites

- Go (1.22.x) // Any version after 1.18.x should be ok, not tested

### How to use

1. Build the backend

```sh
go build backend/main.go
```

2. Run the backend

```sh
# in project root directory if you follow above steps
.\main.exe # assume you are using Windows, otherwise use ./main
```

3. Open the frontend in your browser

```
use any browser you like to open index.html file
```

4. Enjoy it!


### License

This project is licensed under the Do What the Fuck You Want to Public License (WTFPL). This means you can do whatever you want with this project, without any restrictions. For more information, see the `LICENSE` file in the repository.