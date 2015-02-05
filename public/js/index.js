var React = require('react'),
    request = require('superagent'),
    mui = require('material-ui'),
    moment = require('moment');

var Cion = React.createClass({
  loadRepos: function() {
    var url = "/api/" + this.props.owner;
    request
      .get(url)
      .end(function(err, res) {
        if (err || res.error) {
          console.error(url, (err || res.error).toString());
          return;
        }

        this.setState({
          repos: res.body,
        })
      }.bind(this));
  },

  loadJobs: function() {
    var url = "/api/" + this.props.owner + "/" + this.props.repo;
    request
      .get(url)
      .end(function(err, res) {
        if (err || res.error) {
          console.error(url, (err || res.error).toString());
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

var JobDetail = React.createClass({
  loadJob: function() {
    var url = "/api/" + this.props.owner + "/" + this.props.repo + "/" + this.props.number;
    request
      .get(url)
      .end(function(err, res) {
        if (err || res.error) {
          console.error(url, (err || res.error).toString());
          return;
        }

        var job = res.body;
        this.setState({
          job: job,
        });

        if (job && job.EndedAt) {
          clearInterval(this.state.jobInterval);
        }
      }.bind(this));
  },

  componentDidMount: function() {
    this.setState({
      jobInterval: setInterval(this.loadJob, this.props.pollInterval),
    });

    this.loadJob();
  },

  componentDidUpdate: function(prevProps) {
    if (prevProps.owner != this.props.owner ||
        prevProps.repo != this.props.repo ||
        prevProps.number != this.props.number) {
      this.loadJob();
    }
  },

  getInitialState: function() {
    return {
      job: null,
      jobInterval: null,
    };
  },

  render: function() {
    var logUrl = "/api/" + this.props.owner + "/" + this.props.repo + "/" + this.props.number + "/log";
    return (
        <mui.Paper className="jobDetail">
          <pre>
            <iframe className="log" src={logUrl}></iframe>
          </pre>
        </mui.Paper>
    );
  },
});

var JobTable = React.createClass({
  handleSelectJob: function(job) {
    this.setState({
      selectedJob: job,
    });
  },

  getInitialState: function() {
    return {
      selectedJob: null,
    };
  },

  render: function() {
    var jobRowNodes = this.props.jobs.map(function(job) {
      return (
        <JobTable.Row job={job} onClick={this.handleSelectJob.bind(this, job)} />
      )
    }.bind(this));

    var jobDetail = <div></div>;
    if (this.state.selectedJob) {
      jobDetail = <JobDetail
        owner={this.state.selectedJob.Owner}
        repo={this.state.selectedJob.Repo}
        number={this.state.selectedJob.Number}
        pollInterval={5000} />
    }

    return (
      <div>
        <mui.Paper className="jobTable">
          <table>
            <thead>
              <tr>
               <th>#</th>
               <th>Commit</th>
               <th>Started</th>
               <th>Took</th>
              </tr>
            </thead>
            <tbody>
              {jobRowNodes}
            </tbody>
          </table>
        </mui.Paper>

        {jobDetail}
      </div>
    );
  },
});

JobTable.Row = React.createClass({
  render: function() {
    var started = moment(this.props.job.StartedAt).fromNow();

    var ended = this.props.job.EndedAt;
    ended = (ended) ? moment(ended).from(this.props.job.StartedAt, true) : "-";

    return (
      <tr key={this.props.job.Number} className="jobTableRow" onClick={this.props.onClick}>
        <td>{this.props.job.Number}</td>
        <td>{this.props.job.SHA.substring(0, 6)} ({this.props.job.Branch})</td>
        <td>{started}</td>
        <td>{ended}</td>
      </tr>
    );
  },
});

React.render(
  <Cion owner="spotify" repo="docker-client" pollInterval={5000} />,
  document.body
);
