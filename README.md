<p align="center">
  <a href="https://sophon.venus-fil.io/" title="Sophon Docs">
    <img src="https://user-images.githubusercontent.com/1591330/205581370-d467d776-60a4-4b37-b25a-58fa82adb156.png" alt="Sophon Logo" width="128" />
  </a>
</p>

<h1 align="center">sophon-miner</h1>

<p align="center">
  <a href="https://circleci.com/gh/ipfs-force-community/sophon-miner"><img src="https://circleci.com/gh/filecoin-project/venus.svg?style=svg"></a>
  <a href=""><img src="https://img.shields.io/badge/golang-%3E%3D1.18-blue.svg" /></a>
  <br>
</p>

The sophon-miner is used to mine for multiple miners, the advantage is that：
1. The miners in the mining pool do not need to deploy their own mining programs. 
2. It is possible to increase the tps and miner fees for packaging messages.
3. Miners can package messages for each other to avoid the problem of selfish mining.

Use [Venus Issues](https://github.com/filecoin-project/venus/issues) for reporting issues about this repository.

## Building & Documentation

For instructions on how to build, install and setup venus, please visit [https://venus.filecoin.io/](https://venus.filecoin.io/).

## Reporting a Vulnerability

Please send an email to security@filecoin.org. See our [security policy](SECURITY.md) for more details.

## Related packages

These repos are independent and reusable modules, but are tightly integrated into Venus to make up a fully featured Filecoin implementation:
- [specs-actors](https://github.com/filecoin-project/specs-actors) which has its own [kanban work tracker available here](https://app.zenhub.com/workspaces/actors-5ee6f3aa87591f0016c05685/board)

## Contribute

Venus is a universally open project and welcomes contributions of all kinds: code, docs, and more. However, before making a contribution, we ask you to heed these recommendations:

1. If the proposal entails a protocol change, please first submit a [Filecoin Improvement Proposal](https://github.com/filecoin-project/FIPs).
2. If the change is complex and requires prior discussion, [open an issue](https://github.com/ipfs-force-community/sophon-miner/issues) to request feedback before you start working on a pull request. This is to avoid disappointment and sunk costs, in case the change is not actually needed or accepted.
3. Please refrain from submitting PRs to adapt existing code to subjective preferences. The changeset should contain functional or technical improvements/enhancements, bug fixes, new features, or some other clear material contribution. Simple stylistic changes are likely to be rejected in order to reduce code churn.

When implementing a change:

1. Adhere to the standard Go formatting guidelines, e.g. [Effective Go](https://golang.org/doc/effective_go.html). Run `go fmt`.
2. Stick to the idioms and patterns used in the codebase. Familiar-looking code has a higher chance of being accepted than eerie code. Pay attention to commonly used variable and parameter names, avoidance of naked returns, error handling patterns, etc.
3. Comments: follow the advice on the [Commentary](https://golang.org/doc/effective_go.html#commentary) section of Effective Go.
4. Minimize code churn. Modify only what is strictly necessary. Well-encapsulated changesets will get a quicker response from maintainers.
5. Lint your code with [`golangci-lint`](https://golangci-lint.run) (CI will reject your PR if unlinted).
6. Add tests.
7. Title the PR in a meaningful way and describe the rationale and the thought process in the PR description.
8. Write clean, thoughtful, and detailed [commit messages](https://chris.beams.io/posts/git-commit/). This is even more important than the PR description, because commit messages are stored _inside_ the Git history. One good rule is: if you are happy posting the commit message as the PR description, then it's a good commit message.

## License

Dual-licensed under [MIT](https://github.com/ipfs-force-community/sophon-miner/blob/master/LICENSE-MIT) + [Apache 2.0](https://github.com/ipfs-force-community/sophon-miner/blob/master/LICENSE-APACHE)
