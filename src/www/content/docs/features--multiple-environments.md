---
title: Multiple Environments
description: Create multiple development environments easily with Stormkit.
---

# Multiple environments

<section>
With Stormkit you can create multiple environments per application. Each environment will point to a branch and when that branch is updated Stormkit will automatically deploy it (given you have <a href="/docs/deployments/auto-deployments">Auto Deployments</a>.
</section>

# Default environment

<section>
By default, each application comes with a production environment already set. You'll need to configure it to deploy it successfully. The production environment cannot be deleted or renamed. However, you can change the branch it points. Any branch that does not match an environment's branch (a feature branch) will be deployed with the default environment's configuration.

</section>

# Creating an environment

<section>
In order to create a new environment, select your application. You'll be directly brought to a screen where you see a list of environments. On top right, you'll see an <code>Add Environment.</code> Click on that and a <a href="/docs/deployments/configuration">configure</a> your environment.

<div class="img-wrapper">
    <img src="/assets/docs/features/env-screen.png" alt="Env screen" />
</div>

</section>

# Deleting an environment

<section>
<p>
In order to delete an environment, you'll have to navigate to the <a href="/docs/deployments/configuration">configuration</a> page and hit the <b>Delete environment</b> button at the bottom. Deleting an environment will also remove all associated deployments.
</p>
<div>
You cannot delete <b>Production</b> environments as they are required by design.
</div>
</section>
