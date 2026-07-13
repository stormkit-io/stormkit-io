---
title: Multiple Environments
description: Create multiple development environments easily with Stormkit.
---

# Multiple environments

<section>
With Stormkit, you can create multiple environments per application. Each environment points to a specific branch, and when that branch is updated, Stormkit will automatically deploy it (provided you have <a href="/docs/deployments/auto-deployments">Auto Deployments</a> enabled).
</section>

# Default environment

<section>
By default, each application comes with a production environment already set. You'll need to configure it to deploy successfully. The production environment cannot be deleted or renamed, but you can change the branch it points to. Any branch that does not match an environment's configured branch (such as a feature branch) will be deployed using the default environment's configuration.

</section>

# Creating an environment

<section>
To create a new environment, select your application. You'll be taken directly to your application's default environment (production). On the left navigation menu, you'll see an <code>Add Environment</code> button. Click it and then <a href="/docs/deployments/configuration">configure</a> your environment.

<div class="img-wrapper">
    <img src="/assets/docs/features/env-screen.png" alt="Env screen" />
</div>

</section>

# Deleting an environment

<section>
<p>
To delete an environment, navigate to the <a href="/docs/deployments/configuration">configuration</a> page and click the <b>Delete environment</b> button at the bottom. Deleting an environment will also remove all associated deployments.
</p>
<div>
Note: <b>Production</b> environments cannot be deleted as they are required by design.
</div>
</section>
