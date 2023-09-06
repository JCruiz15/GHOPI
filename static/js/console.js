const url = window.location.href.split(location.pathname)[0] + "/api/get-logs";

let filterMenu = document.getElementById("filters");
let logs = document.getElementById("logs");
let startDate = document.getElementById("filterDateStart");
let endDate = document.getElementById("filterDateEnd");
let currentLvl = 4;
let filtered_type = false;
let fileterd_date = false;

async function get_logs() {
    let response = await fetch(url,{
        headers: {Authentication: 'Bearer <token>'} 
    } );
    let data = await response.text();
    return data;
}

async function get_console(lvl) {
    log = await get_logs();
    if(!filtered_type && !filtered_date) {
        return log;
    } else if(filtered_type && !filtered_date && currentLvl < lvl) {
        return log;
    } else if(filtered_type && !filtered_date && currentLvl >= lvl) {
        return logs.innerHTML;
    } else if(filtered_type && filtered_date && currentLvl < lvl) {
        filtered_date = false
        return log;
    } else if(filtered_type && filtered_date && currentLvl >= lvl) {
        return logs.innerHTML;
    } else if(!filtered_type && filtered_date) {
        filtered_date = false
        return log;
    } else {
        return log;
    }
}

async function show_logs() {
    logs.innerHTML = await get_logs();
    filterByType(4);
    filtered_type = false;
    filtered_date = false;
    startDate.value = "";
    endDate.value = "";
}
window.onload = show_logs();

async function filterByType(level) {
    cmd = await get_console(level);
    currentLvl = level;
    let lines = cmd.split("<br>");
    let resul = "";

    for (i=1; i<lines.length-1; i++) {
        l = lines[i].match(/\[([^)]*)\]/)[1];
        switch (level) {
            case 4:
                if(l.includes("INFO") || l.includes("WARNING") || l.includes("ERROR") || l.includes("FATAL")) {
                    resul = resul.concat(lines[i], "<br>");
                }
                break;
            case 3:
                if(l.includes("WARNING") || l.includes("ERROR") || l.includes("FATAL")) {
                    resul = resul.concat(lines[i], "<br>");
                }
                break;
            case 2:
                if(l.includes("ERROR") || l.includes("FATAL")) {
                    resul = resul.concat(lines[i], "<br>");
                }
                break;
            case 1:
                if(l.includes("FATAL")) {
                    resul = resul.concat(lines[i], "<br>");
                }
                break;
        }
    }
    logs.innerHTML = resul;
    filtered_type = true;
    if (!filtered_date && (startDate.value != "" || endDate.value != "")){
        filterByDate()
    }
}

function CheckDate() {
    let start = new Date(startDate.value);
    let end = new Date(endDate.value);
    if (start > end) {
        first = endDate.value;
        endDate.value = startDate.value;
        startDate.value = first;
    }
}

async function filterByDate() {
    all_logs = await get_console(currentLvl);
    let lines = all_logs.split("<br>");
    let resul = "";
    filtered_date = true;

    let start = new Date(startDate.value);
    if(startDate.value === '') {
        start = new Date("1900-01-01T01:00:00");
    }
    let end = new Date(endDate.value);
    if(endDate.value === '') {
        end = new Date("2500-01-01T01:00:00");
    }

    let str1 = '<span id="date" style="color:grey; font-family: monospace;">\t'
    let str2 = '</span>'

    for (i = 0; i < lines.length; i++) {
        date_element_index1 = lines[i].indexOf(str1);   // Gets index of span
        if (date_element_index1 != -1) {
            ind1 = date_element_index1+str1.length;     // If it has de correct format it gets de date between indexes
            ind2 = lines[i].indexOf(str2, ind1);

            date_str = lines[i].substring(ind1 , ind2);
            date = moment(date_str, "DD/MM/YYYY-HH:mm:ss").toDate();
        } else {
            date = new Date("");
        }

        if (date.setHours(23, 59, 59, 999) >= start && date.setHours(0, 0, 0, 0) <= end) { // It'll show dates in between the limits
            resul = resul.concat(lines[i], "<br>");
        }
    }
    logs.innerHTML = resul;
}