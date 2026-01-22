Study the following files in order to understand the project:

1. **[README.md](README.md)** - Project overview, features, and configuration
2. **[DEVELOPERS.md](DEVELOPERS.md)** - Build setup, development workflow, and testing
3. **[docs/README.md](docs/README.md)** - Documentation index and navigation

Pay special attention to instructions about using the Make files!

After you have studied those files, study the [plans](docs/planning/README.md) and read any other docs in the `docs/planning/` folder.  you should find a file `docs/planning/EXECUTION_PLAN.md`.  That file describes the implementation plan for `docs/planning/SPECIFICATION.md`.  If either of those files are missing, STOP!  Tell me which file(s) is/are missing and await further instructions.

Assuming both of those files exist, study them!  Finally run `bd prime`.  This will get you up to speed on where we are, what we are working on and what's left to do.  Some of the work may have been completed in previous sections.  Audit the code against the spec and the plan to deterine what work is left.  

There may be tickets in beads associated with the spec and plan, and you may need to create tickets as you work the plan.  There may be tickets associated with other work, not part of this spec and plan.  You should focus only on tickets and work related to this plan!  

If you find a ticket that is not closed in beads, but you find the work is all the way done, you should check to ensure proper documentation and tests exist for the work, and if everything is perfect you should close that ticket and ensure the plan represents the finished work.  

If there is no work left to do on this spec/plan, then check if the spec has been completely converted to a document (or set of documents) in the docs folder describing the new state of this code.  If that hasn't been done, then converting the spec into documentation is your next task.  

If there are no more related tasks, and the execution plan for this spec is truly done, delete both the `docs/planning/SPECIFICATION.md` and the `docs/planning/EXECUTION_PLAN.md` files, then report back to the user that there's no related work left to do and await further instructions.    

Assuming there still is work left to do on this spec/plan, then do the most imporant thing to move the plan forward.  Pick the one most important next task and do it.  You may use up to 10 subagents in any way you see fit.  If you run into any blockers, try to remove them.  If you cannot remove them, clearly document that you are blocked in `docs/planning/EXECUTION_PLAN.md`.  Also, ensure you tell me you're blocked as well.  Also, you should have a beads task for all your work, so mark your beads task as blocked too.   

Remember to keep the planning docs and beads up to date as you work on the task.  

Remember to dogfood code-scout as you work on the task, it's the best way to learn about this codebase. 

Remember to maintain good documentation quality.  All new features and changes to functionality need to be well documented, in the `docs/` direcotry.  Follow the existing documentation structure.   

Remember to maintain high test coverage.  All new features will need tests.  Bug fixes need tests as well.  

Remember to index with code-scout after your any changes or any new code has been written, tested and documented.  

That is your workflow. Do all these things for the one task you choose.  Only complete these thigns for one task, then report back on the status and await further instructions.  

