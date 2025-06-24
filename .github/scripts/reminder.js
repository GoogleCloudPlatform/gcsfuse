// This script uses the Octokit library to interact with the GitHub API.
const { getOctokit } = require('@actions/github');
// The context object provides information about the workflow run.
const { context } = require('@actions/github');

// This is the main function that will be executed.
async function run() {
    try {
        // ------------------ CONFIGURATION ------------------
        // Label that triggers the reminder.
        const REMINDER_LABEL = 'remind-reviewers';
        // Inactivity time in hours.
        const INACTIVITY_HOURS = 24;
        // The message to post on the pull request.
        const REMINDER_MESSAGE = `Hi {reviewers}, your feedback is needed to move this pull request forward. This automated reminder was triggered because there has been no activity for over ${INACTIVITY_HOURS} hours. Please provide your input when you have a moment. Thank you!`;
        // ---------------------------------------------------

        // Get the GitHub token from environment variables. It's passed in from the workflow file.
        const token = process.env.GITHUB_TOKEN;
        if (!token) {
            throw new Error("GITHUB_TOKEN is not set. The workflow must pass it as an environment variable.");
        }

        // Create an authenticated Octokit client.
        const octokit = getOctokit(token);
        // Remove 10 minutes from inactivity time to not account previous reminders as activity.
        const INACTIVITY_MS = INACTIVITY_HOURS * 60 * 60 * 1000 - 10 * 60 * 1000;
        const now = new Date().getTime();

        // Get all open pull requests
        const pullRequests = await octokit.paginate(octokit.rest.pulls.list, {
            owner: context.repo.owner,
            repo: context.repo.repo,
            state: 'open',
        });

        console.log(`Found ${pullRequests.length} open pull requests.`);

        for (const pr of pullRequests) {
            const hasReminderLabel = pr.labels.some(label => label.name === REMINDER_LABEL);
            if (!hasReminderLabel) {
                console.log(`PR #${pr.number} ('${pr.title}') does not have the '${REMINDER_LABEL}' label. Skipping.`);
                continue;
            }

            if (pr.draft) {
                console.log(`PR #${pr.number} ('${pr.title}') is a draft. Skipping.`);
                continue;
            }

            const updatedAt = new Date(pr.updated_at).getTime();
            const isInactive = (now - updatedAt) > INACTIVITY_MS;
            if (!isInactive) {
                console.log(`PR #${pr.number} ('${pr.title}') is not inactive yet. Last updated at ${pr.updated_at}. Skipping.`);
                continue;
            }

            const requestedReviewers = pr.requested_reviewers.map(reviewer => `@${reviewer.login}`).join(', ');
            if (requestedReviewers.length === 0) {
                console.log(`PR #${pr.number} ('${pr.title}') is inactive but has no requested reviewers. Skipping.`);
                continue;
            }

            const finalMessage = REMINDER_MESSAGE.replace('{reviewers}', requestedReviewers);
            console.log(`PR #${pr.number} ('${pr.title}') is inactive. Posting a reminder comment.`);

            await octokit.rest.issues.createComment({
                owner: context.repo.owner,
                repo: context.repo.repo,
                issue_number: pr.number,
                body: finalMessage
            });
        }
    } catch (error) {
        // If the script fails, log the error message.
        console.error(error.message);
        process.exit(1);
    }
}

// Execute the main function.
run();
