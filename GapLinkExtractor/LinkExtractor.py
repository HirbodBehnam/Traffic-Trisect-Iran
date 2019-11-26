from gapbot import Gap

app = Gap()

@app.on_update
def _update_handler(bot, update):
    if update['type'] == 'file':
        f = open("links.txt", "a")
        f.write(update['data']['path']+"\n")
        f.close()

if __name__ == '__main__':
    app.run()