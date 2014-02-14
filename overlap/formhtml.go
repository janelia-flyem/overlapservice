package overlap

const formHTML=`
<html>                                                                                                   
<head>
<script src="http://code.jquery.com/jquery-1.11.0.min.js"></script>                                      
<script type="text/javascript" src="//www.google.com/jsapi"></script>
<script type="text/javascript">
    google.load('visualization', '1.1', {packages: ['controls']});
</script>

</head>
<H2>Calculate Overlap and Body Stats</H2>
<form id="overlapcalc" method="post">
<input type="radio" name="group1" id="omode" value="overlap" checked>Calculate Overlap<br>
<input type="radio" name="group1" id="smode" value="stats">Calculate Body Stats<br>
DVID server (e.g., emdata1:80): <input type="text" id="dvidserver" value="DEFAULT"><br>
DVID uuid: <input type="text" id="uuid"><br>
Body list (e.g., 3, 4, 34): <input type="text" id="bodies"><br>
<input type="submit" value="Submit"/>
</form>

<hr>
<br>
<div id="status"></div><br>
<div id="dashboard">
      <table>
        <tr style='vertical-align: top'>
          <td style='width: 300px; font-size: 0.9em;'>
            <div id="control1"></div>
            <div id="control2"></div>
          </td>
          <td style='width: 600px'>
            <div style="float: left;" id="chart1"></div>
          </td>
        </tr>
      </table>
</div>
<div id="bdashboard">
      <table>
        <tr style='vertical-align: top'>
          <td style='width: 300px; font-size: 0.9em;'>
            <div id="bcontrol1"></div>
          </td>
          <td style='width: 600px'>
            <div style="float: left;" id="bchart1"></div>
          </td>
        </tr>
      </table>
</div>

<script>
    $("#overlapcalc").submit(function(event) {                                                           
      event.preventDefault();
      var querytype = $("input[name='group1']:checked").val()
      $('#status').html("Processing " + querytype + "....");
      $('#dashboard').hide();  
      $('#bdashboard').hide();  

      if (querytype == "overlap") {
          $.ajax({
            type: "POST",
            url: "/formhandler/",
            data: {uuid: $('#uuid').val(), bodies: $('#bodies').val(), dvidserver: $('#dvidserver').val()},
            success: function(data){
                var results = data["overlap-list"];
                    
                if (results == "") {
                    $('#status').html("No overlaps exist");
                    return;
                }
                $('#status').html("")

                var column_names = ["Body 1", "Body 2", "overlap (# touching voxel faces)"];
                var data_rows = [column_names];

                for (var i in results) {
                        var result_obj = results[i]
                        var rowval = [result_obj[0], result_obj[1], result_obj[2]];
                        data_rows.push(rowval);
                }


                // Prepare the data.
                var data = google.visualization.arrayToDataTable(data_rows);
              
                // Define a StringFilter control for the 'bodyid' column
                var stringFilter = new google.visualization.ControlWrapper({
                  'controlType': 'StringFilter',
                  'containerId': 'control1',
                  'options': {
                    'filterColumnLabel': 'Body 1',
                    'ui' : {
                        'label': 'Body 1 Filter'
                    }
                  }
                });

                // Define a StringFilter control for the 'bodyid' column
                var stringFilter2 = new google.visualization.ControlWrapper({
                  'controlType': 'StringFilter',
                  'containerId': 'control2',
                  'options': {
                    'filterColumnLabel': 'Body 2',
                    'ui' : {
                        'label': 'Body 2 Filter'
                    }
                  }
                });
              
                // Define a table visualization
                var table = new google.visualization.ChartWrapper({
                  'chartType': 'Table',
                  'containerId': 'chart1',
                  'options': {'height': '13em', 'width': '25em'}
                });
              
                // Create the dashboard.
                var dashboard = new google.visualization.Dashboard(document.getElementById('dashboard')).
                  // Configure the string filter to affect the table contents
                  bind(stringFilter, table).
                  bind(stringFilter2, table).
                  // Draw the dashboard
                  draw(data);

                $('#dashboard').show();  
              },

            error: function(msg) {
                    $('#status').html("Error Processing Results: " + msg.responseText)
              }
            });
        } else {
            $.ajax({
            type: "POST",
            url: "/formhandler2/",
            data: {uuid: $('#uuid').val(), bodies: $('#bodies').val(), dvidserver: $('#dvidserver').val()},
            success: function(data){
                var results = data["body-stats"];
                    
                if (results == "") {
                    $('#status').html("No body stats found");
                    return;
                }
                $('#status').html("")

                var column_names = ["Body", "Volume (in voxels)", "Surface Area (# voxel faces)"];
                var data_rows = [column_names];

                for (var i in results) {
                        var result_obj = results[i]
                        var rowval = [result_obj[0], result_obj[1], result_obj[2]];
                        data_rows.push(rowval);
                }

                // Prepare the data.
                var data = google.visualization.arrayToDataTable(data_rows);
              
                // Define a StringFilter control for the 'bodyid' column
                var stringFilter = new google.visualization.ControlWrapper({
                  'controlType': 'StringFilter',
                  'containerId': 'bcontrol1',
                  'options': {
                    'filterColumnLabel': 'Body',
                    'ui' : {
                        'label': 'Body Filter'
                    }
                  }
                });

                // Define a table visualization
                var table = new google.visualization.ChartWrapper({
                  'chartType': 'Table',
                  'containerId': 'bchart1',
                  'options': {'height': '13em', 'width': '30em'}
                });
              
                // Create the dashboard.
                var dashboard = new google.visualization.Dashboard(document.getElementById('bdashboard')).
                  // Configure the string filter to affect the table contents
                  bind(stringFilter, table).
                  // Draw the dashboard
                  draw(data);

                $('#bdashboard').show();  
              },

            error: function(msg) {
                    $('#status').html("Error Processing Results: " + msg.responseText)
              }
            });
        }
    });                                                                                                  
</script>                                                                                                
</html>                                  
`
