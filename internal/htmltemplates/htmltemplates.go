package htmltemplates

var SubmitTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Seilfahrt</title>
</head>
<body>
    <h1>Enter a URL</h1>
    <form action="/submit" method="POST">
        <label for="url">Hedgedoc Plenumsprotokoll URL:</label>
        <input type="text" id="url" name="url" required>
        <button type="submit">Submit</button>
    </form>
</body>
</html>
`

var ResponseTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Seilfahrt </title>
</head>
<body>
    <h1>{{.Title}}</h1>
    <p>{{.Message}}</p>
    <a href="/">Go back</a>
</body>
</html>
`
