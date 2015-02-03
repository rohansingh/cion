var React = require('react'),
    request = require('superagent'),
    mui = require('material-ui');

var Cion = React.createClass({
  loadRepos: function() {
    request
      .get("/api/" + this.props.owner)
      .end(function(err, res) {
        if (err || res.error) {
          console.err(this.props.url, (err || res.error).toString());
          return;
        }

        this.setState({
          repos: res.body,
        })
      }.bind(this));
  },

  loadJobs: function() {
    request
      .get("/api/" + this.props.owner + "/" + this.props.repo)
      .end(function(err, res) {
        if (err || res.error) {
          console.err(this.props.url, (err || res.error).toString());
          return;
        }

        this.setState({
          jobs: res.body,
        });
      }.bind(this));
  },

  componentDidMount: function() {
    this.loadRepos();
    this.loadJobs();

    setInterval(this.loadRepos, this.props.pollInterval);
    setInterval(this.loadJobs, this.props.pollInterval);
  },

  getInitialState: function() {
    return {
      repos: [],
      jobs: [],
    };
  },

  render: function() {
    return (
      <div className="cion">
        <div className="sidebar">
          <Sidebar owner={this.props.owner} repos={this.state.repos} currentRepo={this.props.repo} />
        </div>

        <div className="content">
          <Repo owner={this.props.owner} repo={this.props.repo} jobs={this.state.jobs} />
        </div>
      </div>
    );
  },
});

var Sidebar = React.createClass({
  render: function() {
    var menuItems = [{
      type: mui.MenuItem.Types.SUBHEADER,
      text: this.props.owner,
    }];

    var selectedIndex = null;

    this.props.repos.map(function(r) {
      if (r == this.props.currentRepo) {
        selectedIndex = menuItems.length;
      }

      menuItems.push({
        route: this.props.owner + '/' + r,
        text: r,
      });
    }.bind(this));

    return (
      <mui.Menu menuItems={menuItems} selectedIndex={selectedIndex} />
    );

  },
});

var Repo = React.createClass({
  render: function() {
    return (
      <div className="repo">
        <JobTable jobs={this.props.jobs} />
      </div>
    );
  },
});

var JobTable = React.createClass({
  render: function() {
    var jobRowNodes = this.props.jobs.map(function (job) {
      return (
        <JobTable.Row job={job} />
      )
    });

    return (
      <table className="jobTable">
        <thead>
          <tr>
           <th>#</th>
           <th>Commit</th>
           <th>Started</th>
           <th>Ended</th>
          </tr>
        </thead>
        {jobRowNodes}
      </table>
    );
  },
});

JobTable.Row = React.createClass({
  render: function() {
    var ended = this.props.job.EndedAt || "-";
    return (
      <tr key={this.props.job.Number} className="jobTableRow">
        <td>{this.props.job.Number}</td>
        <td>{this.props.job.SHA} ({this.props.job.Branch})</td>
        <td>{this.props.job.StartedAt}</td>
        <td>{ended}</td>
      </tr>
    );
  },
});

React.render(
  <Cion owner="spotify" repo="docker-client" pollInterval={2000} />,
  document.body
);
