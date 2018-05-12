function onCall(number) {
    if (typeof number !== "string") {
        return "";
    }
    if (/\+861\d{10}/.test(number)) {
        return "15606568421," + number.substr(3);
    }
    if (/\+861[\d ]{10,13}/.test(number)) {
        var new_number = number.substr(3);
        new_number = new_number.replace(" ", "");
        return "15606568421," + new_number;
    }
    if (/1[\d ]{10,13}/.test(number)) {
        return "15606568421," + number.replace(" ", "");
    }
    return number;
}

function onCall(number) {
    if (number.size < 4) {
        return number
    }
    if (number.substr(0, 4) === "+861") {

    }
}