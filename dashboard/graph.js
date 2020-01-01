var data = {"nodes":[]};

function toString(config) {
    return config["ip"] + ":" + config["port"].toString();
}

function fetchData(url, callback) {
    var xmlhttp = new XMLHttpRequest();
    xmlhttp.open("GET", url+"/peers", null);
    xmlhttp.onreadystatechange = function() {
        if (xmlhttp.readyState == 4) {
            if(xmlhttp.status == 200) {
                console.log(xmlhttp.responseText);
                data = JSON.parse(xmlhttp.responseText);
                callback(data);
            }
        }
    }
    xmlhttp.send();
}

function parseData(data) {
    var nodes = [];
    var edges = [];
    var analyzed = [];

    if (data["nodes"] !== null) {
        data["nodes"].forEach(function(node, i, _) {
            config = toString(node["config"]);

            nodes.push({
                id: config,
                label: config
            });

            if (node["peers"] !== null) {
                node["peers"].forEach(function(peer, j, _) {
                    if (!analyzed.includes(toString(peer))) {
                        edges.push({
                            from: config,
                            to: toString(peer)
                        });
                    }
                });
            }

            analyzed.push(toString(node["config"]));
        });
    }

    return {
        nodes: new vis.DataSet(nodes),
        edges: new vis.DataSet(edges)
    };
}

function createGraph(data) {
    var container = document.getElementById("graph");
    var options = {
        nodes: {
            color: {
                border: "#0C9B3C",
                background: "#52CE7B",
                highlight: {
                    background: "#0C9B3C"
                }
            }
        },
        edges: {
            color: {
                color: "#0C9B3C"
            }
        }
    };
    var network = new vis.Network(container, data, options);
}

document.getElementById("ctrlUpdate").onclick = function() {
    var url = document.getElementById("ctrlAddr").value;
    fetchData(url, function(data) {
        createGraph(parseData(data));
    });
}