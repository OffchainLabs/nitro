# Contribution Guidelines

Note: The latest and most up-to-date documentation can be found on our [docs portal](https://docs.arbitrum.io/launch-arbitrum-chain/a-gentle-introduction).

Excited by our work want to get more involved in making Arbitrum more successful? Or maybe you want to learn more about Layer 2 technologies and want to contribute as a first step?

You can explore our [Open Issues](https://github.com/offchainlabs/nitro/issues) or [run a Nitro node](https://docs.arbitrum.io/run-arbitrum-node/run-nitro-dev-node) yourself and suggest improvements. 

<!-- start-trivial-prs -->
> [!IMPORTANT] 
> Please, **do not send pull requests for trivial changes**; these will be rejected.
> These types of pull requests incur a cost to reviewers and do not provide much value to the project.
> If you are unsure, please open an issue first to discuss the change.
> Here are some examples of trivial PRs that will most-likely be rejected:
> * Fixing typos
> * AI-generated code
> * Refactors that don't improve usability
<!-- end-trivial-prs -->

## Contribution Steps

**1. Build Nitro locally following our instructions in our [docs](https://docs.arbitrum.io/run-arbitrum-node/nitro/build-nitro-locally).**

**2. Fork the Nitro repo.**

Sign in to your GitHub account or create a new account if you do not have one already. Then navigate your browser to https://github.com/offchainlabs/nitro. In the upper right hand corner of the page, click “fork”. This will create a copy of the Nitro repo in your account.

**3. Create a local clone of Nitro.**

```
$ git clone https://github.com/OffchainLabs/nitro.git
```

**4. Link your local clone to the fork on your GitHub repo.**

```
$ git remote add mynitrorepo https://github.com/<your_github_user_name>/nitro.git
```

**5. Link your local clone to the Nitro repo so that you can easily fetch future changes.**

```
$ git remote add upstream https://github.com/offchainlabs/nitro.git
$ git remote -v (you should see mynitrorepo and upstream in the list of remotes)
```

**6. Create a local branch with a name that clearly identifies what you will be working on.**

```
$ git checkout -b feature-in-progress-branch
```

**7. Make improvements to the code.**

Each time you work on the code be sure that you are working on the branch that you have created as opposed to your local copy of the Nitro repo. Keeping your changes segregated in this branch will make it easier to merge your changes into the repo later.

```
$ git checkout feature-in-progress-branch
```

**8. Test your changes.**

Write unit tests or write a [system test](https://github.com/OffchainLabs/nitro/tree/master/system_tests) for your feature before shipping it.

**9. Stage the file or files that you want to commit.**

```
$ git add --all
```

This command stages all the files that you have changed. You can add individual files by specifying the file name or names and eliminating the “-- all”.

**10. Commit the file or files.**

```
$ git commit  -m “Message to explain what the commit covers”
```

You can use the –amend flag to include previous commits that have not yet been pushed to an upstream repo to the current commit. Ensure commit messages are informative and provide sufficient context about your edits.

**11. Fetch any changes that have occurred in the upstream Nitro repo since you started work.**

```
$ git fetch upstream
```

**12. Push your changes to your fork of the Nitro repo.**

Use git push to move your changes to your fork of the repo.

```
$ git push mynitrorepo feature-in-progress-branch
```

**13. Create a pull request.**

Navigate your browser to https://github.com/offchainlabs/nitro and click on the new pull request button. In the “base” box on the left, leave the default selection “base master”, the branch that you want your changes to be applied to. In the “compare” box on the right, select feature-in-progress-branch, the branch containing the changes you want to apply. 

**14. Respond to comments by Core Contributors.**

Core Contributors may ask questions and request that you make edits. If you set notifications at the top of the page to “not watching,” you will still be notified by email whenever someone comments on the page of a pull request you have created. If you are asked to modify your pull request, repeat steps 8 through 15, then leave a comment to notify the Core Contributors that the pull request is ready for further review.

**15. If the number of commits becomes excessive, you may be asked to squash your commits.**

 You can do this with an interactive rebase. Start by running the following command to determine the commit that is the base of your branch...

```
$ git merge-base feature-in-progress-branch nitro/master
```

**16. The previous command will return a commit-hash that you should use in the following command.**

```
$ git rebase -i commit-hash
```

Your text editor will open with a file that lists the commits in your branch with the word pick in front of each branch such as the following …

```
pick 	hash	do some work
pick 	hash 	fix a bug
pick 	hash 	add a feature
```

Replace the word pick with the word “squash” for every line but the first, so you end with ….

```
pick    hash	do some work
squash  hash 	fix a bug
squash  hash 	add a feature
```

Save and close the file, then a commit command will appear in the terminal that squashes the smaller commits into one. Check to be sure the commit message accurately reflects your changes and then hit enter to execute it.

**17. Update your pull request with the following command.**

```
$ git push mynitrorepo feature-in-progress-branch
```

**18.  Finally, again leave a comment to the Core Contributors on the pull request to let them know that the pull request has been updated.**

We love working with people that are autonomous, bring new experience to the team, and are excited for their work. 

Join our dynamic team of innovators and explore exciting career opportunities below to make a meaningful impact in a collaborative environment. Browse open positions and take the next step in your career today!

[Offchain Labs Careers](https://www.offchainlabs.com/careers)
