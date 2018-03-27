var lastUpdate = Date.now();
var totalPythonRequests = 0;
var totalNatsStreamingRequests = 0;

var pythonKey = "www.solo.io;spike";
var natsStreamingRequestsKey = "www.solo.io;nats-streaming-spike";

function loaded() {
    post_analytics();
    update_analytics();
    updatetime();
}

function send_analytics(num) {
    for (let i = 0; i < num; i++) {
        post_analytics_for("spike");
        totalPythonRequests += 1;
    }
}

function send_nats_analytics(num) {
    for (let i = 0; i < num; i++) {
        post_nats_analytics_for("nats-streaming-spike");
        totalNatsStreamingRequests += 1;
    }
}

function update_analytics() {
    $.getJSON( "/analytics", function( data ) {
        lastUpdate = Date.now();
        var items = [];
        $.each( data, function( key, val ) {
            /*
            if (key == pythonKey) {
                val = val + " / " + totalPythonRequests;
            }
            if (key == natsStreamingRequestsKey) {
                val = val + " / " + totalNatsStreamingRequests;
            }
            */
            items.push( "<li id='" + key + "'>" + key + ": " + val + "</li>" );
        });
        $("#currentstats").empty();

        $( "<ul/>", {
            "class": "my-new-list",
            html: items.join( "" )
        }).appendTo( "#currentstats" );
        }).always(getagain);
}

function getagain() {
    setTimeout(update_analytics, 1000);
}

var maxTimeInSecinds = 0;

function updatetime() {
    setTimeout(updatetime, 1000);
    if (myChart) {
        updateChart(1000);
    }
}

function post_analytics() {
    return post_analytics_for(window.location.pathname)    
}

function post_analytics_for(page) {
    return post_analytics_with_address_for("/analytics", page);
}

function post_nats_analytics_for(page) {
    return post_analytics_with_address_for("/analytics-nats", page);
}

var requestId = 0;

var requestStartTimes = {};
var requestLatencies = [];

function post_analytics_with_address_for(address, page) {
    let copyRequestId = requestId;
    requestId += 1;

    requestStartTimes[copyRequestId] = Date.now();

    let data = {
        "Url": "www.solo.io",
        "Page": page,
    }
    readquestStarted(copyRequestId);
    return $.ajax({
        type: "POST",
        url: address,
        data: JSON.stringify(data),
        contentType: 'application/json; charset=utf-8',
        }).fail(function(jqXHR, textStatus) {
            readquestFinished(copyRequestId, false);
        }
        ).done(function(msg) {
            readquestFinished(copyRequestId, true);
        }
        );

}

var pendingRequests = 0;
var succesfulRequests = 0;
var failedRequests = 0;

function readquestStarted(requestId) {
    pendingRequests += 1;
}

function readquestFinished(requestId, successfully) {
    let starttime = requestStartTimes[requestId];
    delete requestStartTimes[requestId];
    let endtime = Date.now();
    pendingRequests -= 1;
    if (successfully) {
        succesfulRequests += 1;
    } else {
        failedRequests += 1;
    }
}

var ctx = null;
var myChart = null;

$(document).ready(function(){
    ctx = $("#myChart");

    myChart = new Chart(ctx, {
  type: 'line',
  data: {
    labels: [0],
    datasets: [{
        data: [0],
        lineTension: 0,
        backgroundColor: 'transparent',
        borderColor: '#007bff',
        borderWidth: 4,
        pointBackgroundColor: '#007bff'
      },{
        data: [0],
        lineTension: 0,
        backgroundColor: 'transparent',
        borderColor: '#ff0000',
        borderWidth: 4,
        pointBackgroundColor: '#ff0000',
        yAxisID : "persecond"
      },{
        data: [0],
        lineTension: 0,
        backgroundColor: 'transparent',
        borderColor: '#00ff00',
        borderWidth: 4,
        pointBackgroundColor: '#00ff00',
        yAxisID : "persecond"
      }]
  },
  options: {
    scales: {
      yAxes: [{
        ticks: {
          beginAtZero: true,
          suggestedMax: 2000
        }
      },
      {
        id : "persecond",
        position: "right",
        ticks: {
            beginAtZero: true,
            stepSize : 1.0
        }
      }
      ]
    },
    legend: {
      display: false,
    }
  }
});
});

function updateChart(dt) {
    dt = dt / 1000.0;
    label = parseFloat(myChart.data.labels[myChart.data.labels.length-1]) + dt;
    myChart.data.labels.push(label);

    myChart.data.datasets[0].data.push(pendingRequests);
    myChart.data.datasets[1].data.push(failedRequests);
    myChart.data.datasets[2].data.push(succesfulRequests);
    succesfulRequests = 0;
    failedRequests = 0;

    if (myChart.data.labels.length > 500) {
        myChart.data.labels.shift();
        myChart.data.datasets.forEach((dataset) => {
            dataset.data.shift();
        });
    }
    

    myChart.update();
}