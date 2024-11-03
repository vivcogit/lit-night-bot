package bot

const BooksPerPage = 5

const DATE_LAYOUT = "02.01.2006"

const removePrefix = "❌"
const choosePrefix = "☑️"

const callbackParamsDelimeter = ":"

var ProgressJokes = []string{
	"Думаю... думаю... кажется, нашел книгу, которая смотрит на меня в ответ!",
	"Дайте мне пару секунд, книги устраивают бой за ваше внимание!",
	"Секунду... мне нужно спросить у всех персонажей, кто готов к встрече.",
	"Так, так, так... какая из книг станет звездой сегодняшнего вечера?",
	"Магический шар предсказаний вращается... и... почти готов!",
	"Дайте мне немного времени, книги все еще спорят, кто из них лучше.",
	"Выбираю... кажется, одна из книг шепчет мне на ухо!",
	"Загружаю данные... о, кажется, одна книга уже подмигнула мне!",
	"Книги так и прыгают на полку, дайте мне минутку их успокоить!",
	"Книги играют в прятки, но я вот-вот их найду!",
	"Загружаю данные... 99%, 99%, 99%... Ой, опять зависло на 99%.",
	"Ищу ответ в 42-страничной инструкции... почти нашел!",
	"Анализирую отражение луны в глазах Шрека.",
	"Отправляю запрос котам. Ответ от кота: 'мяу'. Перевожу...",
	"Провожу многоходовочку, как Шерлок в последнем сезоне.",
	"Запускаю квантовый процессор. Ой, это тостер, секунду...",
	"Пересчитываю уток в Майнкрафт... Подождите немного.",
	"Секунду... нахожу нужную инфу в файлах 'Мемы 2012-го'.",
	"Проверяю, хватит ли колбасы для этого вычисления.",
	"Сейчас, обнуляю счетчик дня без багов... Ой, он снова сломался.",
}

const setDeadlineRequestMessage = "Пожалуйста, ответь на это сообщение, указав новую дату дедлайна в формате дд.мм.гггг. Например, 11.02.2024. Жду твоего ответа!"

const addBooksToWishlistRequestMessage = "Чтобы добавить книги в вишлист, ответь на это сообщение, написав список книг. Каждую книгу, пожалуйста, укажи с новой строки. 😊"

const menuText = "Выберите действие"
