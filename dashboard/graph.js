var colors = {
    "main": "#5F84F4",
    "accent": "#314275",
    "text": "#FFFFFF"
};

function toString(addr) {
    return addr["ip"] + ":" + addr["port"].toString();
}

function fetchData(url, callback) {
    var xhr = new XMLHttpRequest();
    xhr.open("GET", url+"/peers", null);
    xhr.onreadystatechange = function() {
        if (xhr.readyState == 4) {
            if(xhr.status == 200) {
                console.log(xhr.responseText);
                data = JSON.parse(xhr.responseText);
                callback(data);
            }
        }
    }
    xhr.send();
}

function parseData(data) {
    var nodes = [];
    var analyzed = [];

    if (data["nodes"] !== null) {
        data["nodes"].forEach(function(node, i, _) {
            addr = toString(node["addr"]);

            nodes.push({data: {
                id: addr
            }});

            if (node["peers"] !== null) {
                node["peers"].forEach(function(peer, j, _) {
                    if (!analyzed.includes(toString(peer))) {
                        nodes.push({data:{
                            id: addr+"-"+toString(peer),
                            source: addr,
                            target: toString(peer)
                        }});
                    }
                });
            }

            analyzed.push(toString(node["addr"]));
        });
    }

    return nodes;
}

window.onload = function() {
    fetchData(document.getElementById("ctrlAddr").value, function(data) {
        data = parseData(data);
        var cy = cytoscape({
            container: document.getElementById('graph'),
            elements: data,
            style: [
                {
                    selector: 'node',
                    style: {
                        'background-color': colors.main,
                        'label': 'data(id)',
                        'font-family': 'Montserrat'
                    }
                },
                {
                    selector: 'edge',
                    style: {
                        'width': 2,
                        'line-color': colors.accent
                    }
                }
            ],
            layout: {
                name: "circle"
            }
          });
    });

    document.getElementById("ctrlUpdate").onclick = function() {

    }
};