## Contributing to ipmi_exporter

In the spirit of [free software][free-sw], **everyone** is encouraged to help
improve this project. Here are some ways that *you* can contribute:

* Use alpha, beta, and pre-released software versions.
* Report bugs.
* Suggest new features.
* Write or edit documentation.
* Write specifications.
* Write code; **no patch is too small**: fix typos, add comments, add tests,
  clean up inconsistent whitespace.
* Refactor code.
* Fix [issues][].
* Review patches.

## Submitting an issue

We use the [GitHub issue tracker][issues] to track bugs and features. Before
you submit a bug report or feature request, check to make sure that it has not
already been submitted. When you submit a bug report, include a [Gist][] that
includes a stack trace and any details that might be necessary to reproduce the
bug.

## Submitting a pull request

1. [Fork the repository][fork].
2. [Create a topic branch][branch].
3. Implement your feature or bug fix.
4. Unlike we did so far, maybe add tests and make sure that they completely
   cover your changes and potential edge cases.
5. If there are tests now, run `go test`. If your tests fail, revise your code
   and tests, and rerun `go test` until they pass.
6. Add documentation for your feature or bug fix in the code, documentation, or
   PR/commit message.
7. Commit and push your changes.
8. [Submit a pull request][pr] that includes a link to the [issues][] for which
   you are submitting a bug fix (if applicable).

<!-- Alphabetize list: -->
[branch]: http://learn.github.com/p/branching.html
[fork]: http://help.github.com/fork-a-repo
[free-sw]: http://www.fsf.org/licensing/essays/free-sw.html
[gist]: https://gist.github.com
[issues]: https://github.com/prometheus-community/ipmi_exporter/issues
[pr]: http://help.github.com/send-pull-requests
