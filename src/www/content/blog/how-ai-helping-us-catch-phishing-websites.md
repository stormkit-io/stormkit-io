---
title: How AI helps us catching phishing websites
date: 2023-09-26
description: This post explores how Stormkit combats phishing on their hosting platform. They initially used a javascript program for detection, but faced false positives and evasion tactics. By integrating AI through Teachable Machine and TensorFlow.js, they markedly improved accuracy.
---

In the field of hosting services, we all face a common challenge: dealing with malicious users who exploit our platforms for phishing activities. Stormkit, like other providers, also encounters these malicious actors. In this blog post, we will share some of our strategies for identifying and preventing phishing attempts, including the use of simple AI technology.

We first introduced a simple javascript program that is triggered whenever there is a new deployment in our system and we check the newly deployed website for signs of phishing content. This program scans the website's content and checks known phishing patterns and indicators. After that, the score is calculated. If the score is more than threshold an alert is triggered, and we get notification in Discord.

To give you a better idea, let's consider an instance where the program identifies two text fields designated for login, along with a keyword such as 'Facebook'. This scenario raises suspicions that the website may be attempting to mimic Facebook's login interface. Another noteworthy example is when the page incorporates a URL previously reported to Google, further contributing to the overall score that ultimately prompts an alert through Discord.


<div class="img-wrapper">
    <img src="/assets/blog/ai.png" alt="AI system warning us when there is a potential fraud for human validation" />
</div>

This method of using a simple javascript program to scan newly deployed websites for signs of phishing content helps to a certain extent. However, it has introduced two problems that we need to address. Firstly, we have started to encounter false positives. This is something we are okay with because it's somewhat better than manual labor. Secondly, we have noticed that some of the phishing websites we observe do not follow the patterns we have hard coded into our detection system. This means that our current system might miss certain phishing attempts, allowing them to go undetected.

To strengthen our defense against phishing, we have decided to leverage the potential of AI to assist us in this endeavor. Analyzing the contents of HTML alone is not sufficient. We needed a more human-like approach, so we decided to take screenshots of websites and feed them into our pre-trained model, combining this with our existing approach. Since we are not experts in the field of AI, we researched how we could achieve this quickly and came across two tools that are quite helpful: Teachable Machine and TensorFlow.js.

[Teachable Machine](https://teachablemachine.withgoogle.com/) is an incredible tool powered by [web assembly](https://webassembly.org/) and TensorFlow.js. It enables us to train models directly in our browser using data. Whenever we encountered phishing content, we took a screenshot and fed it into our model. We tried to train our models with phishing websites that resembled login pages, as well as other sites that looked different. Essentially, our model has two classes. Pages that look like login pages and pages that do not look like a login page.
[TensorFlow.js](https://www.tensorflow.org/js) is an open-source library that allows for the execution of machine learning and deep learning models directly in the browser or in Node.js environments. It is an extension of the widely used TensorFlow library, which is used for building and training machine learning models. We can download trained models from Teachable Machine and load it in our Node.js program using the same TensorFlow.js library.
This improved our detection accuracy, but we still face the challenge of identifying phishing sites that have a different appearance from login pages. To reduce the amount of false positives that we encounter we added a list of trusted user’s to our system which is excluded from the scanning process.

Our battle against phishing on Stormkit has witnessed significant progress. We initiated with a javascript program, but encountered challenges like false positives and sophisticated evasion techniques. Subsequently, we integrated AI capabilities through tools like Teachable Machine and TensorFlow.js, markedly enhancing our detection capabilities. While refinement is underway, especially in identifying less conventional phishing sites, we're actively addressing the issue. We're committed to ongoing improvement and look forward to sharing further details in our pursuit of a more secure platform. Stay tuned for updates on our progress.


