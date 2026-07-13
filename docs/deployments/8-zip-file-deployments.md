---
title: Deploying a Bare App Using Zip Files
description: This document provides a step-by-step guide on creating and deploying a bare app on Stormkit using zip files. It covers prerequisites, app creation, configuration, zip file preparation, deployment, and troubleshooting tips.
---

# Deploying a Bare App Using Zip Files

This guide explains how to create a bare app on Stormkit and deploy it using zip files.

## Prerequisites

- A zip file containing your app's build files (e.g., `index.html`, `style.css`, etc.).

## Steps to Create and Deploy a Bare App

### 1. Create a New App

1. Log in to your Stormkit dashboard.
2. Click on the **Create new app** button.
3. Select **Create bare app**.

### 2. Configure the App

1. Navigate to the app's **Config** tab.
2. Set up your app's environment variables, domains, and other configurations as needed.

<div class="blog-alert">

> **Note**: If you specify a server command before uploading the zip file, the deployment will be treated as a server deployment instead of a static deployment.

</div>

### 3. Prepare Your Zip File

1. Ensure your app's build files are ready and organized in a folder.

2. Compress the folder into a `.zip` file. For example:
   ```bash
   zip -r app-build.zip /path/to/your/build/files
   ```

### 4. Deploy the Zip File

1. Now click the **Deploy** button to open the modal.
2. Select your zip file and upload it.
3. Click **Deploy now** to proceed.

### 5. Verify the Deployment

1. After the deployment is complete click the **Preview** button.
2. Open the provided URL to verify that your app is live.
