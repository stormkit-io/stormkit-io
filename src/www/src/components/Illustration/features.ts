import imgDeployments from '~/assets/images/screenshots/deployments.png'
import imgDeploymentsL from '~/assets/images/screenshots/deployments_light.png'
import imgEnvironments from '~/assets/images/screenshots/environments.png'
import imgEnvironmentsL from '~/assets/images/screenshots/environments_light.png'
import imgConfiguration from '~/assets/images/screenshots/config.png'
import imgConfigurationL from '~/assets/images/screenshots/config_light.png'
import imgApplications from '~/assets/images/screenshots/apps.png'
import imgApplicationsL from '~/assets/images/screenshots/apps_light.png'
import imgActivityFeed from '~/assets/images/screenshots/activity_feed.png'
import imgActivityFeedL from '~/assets/images/screenshots/activity_feed_light.png'
import imgAnalytics from '~/assets/images/screenshots/analytics.png'
import imgAnalyticsL from '~/assets/images/screenshots/analytics_light.png'
import imgStatusChecks from '~/assets/images/screenshots/status_checks.png'
import imgStatusChecksL from '~/assets/images/screenshots/status_checks_light.png'
import imgPeriodicTriggers from '~/assets/images/screenshots/triggers.png'
import imgPeriodicTriggersL from '~/assets/images/screenshots/triggers_light.png'
import imgVolumes from '~/assets/images/screenshots/volumes.png'
import imgVolumesL from '~/assets/images/screenshots/volumes_light.png'
import imgSnippets from '~/assets/images/screenshots/snippets.png'
import imgSnippetsL from '~/assets/images/screenshots/snippets_light.png'

export interface FeatureProps {
  title: string
  image: string
  imageLight: string
  description: string
  id: string
  maxWidth?: any
  arrow?: {
    left?: any
    top?: any
  }
  pos?: {
    x: any
    y: any
  }
}

export const features: FeatureProps[] = [
  {
    title: 'Deployments',
    image: imgDeployments,
    imageLight: imgDeploymentsL,
    description:
      'Instantly deploy your applications with automatic CI/CD pipelines',
    id: 'feature-deployments',
    maxWidth: 300,
    arrow: {
      left: '50%',
      top: '100%',
    },
    pos: {
      x: { md: 370 },
      y: { md: 70 },
    },
  },
  {
    title: 'Applications',
    image: imgApplications,
    imageLight: imgApplicationsL,
    description:
      'Connect your git repository and manage multiple applications from a centralized dashboard seamlessly',
    id: 'feature-applications',
    maxWidth: 300,
    pos: {
      x: { md: 850 },
      y: { md: 140 },
    },
    arrow: {
      left: '0',
      top: '50%',
    },
  },
  {
    title: 'Environments',
    image: imgEnvironments,
    imageLight: imgEnvironmentsL,
    description:
      'Create development environments for your apps and configure them separately',
    id: 'feature-environments',
    arrow: {
      left: '50%',
      top: '100%',
    },
    pos: {
      x: { md: 76 },
      y: { md: 90 },
    },
  },
  {
    title: 'Activity feed',
    image: imgActivityFeed,
    imageLight: imgActivityFeedL,
    description: 'Track team actions with a comprehensive event timeline',
    id: 'feature-activity-feed',
    pos: {
      x: { md: 450 },
      y: { md: 155 },
    },
    arrow: {
      left: '0',
    },
  },
  {
    title: 'Analytics',
    image: imgAnalytics,
    imageLight: imgAnalyticsL,
    description:
      'Built-in, privacy-friendly analytics for your web applications',
    id: 'feature-analytics',
    maxWidth: 300,
    pos: {
      x: { md: 410 },
      y: { md: 175 },
    },
    arrow: {
      left: '0',
      top: '50%',
    },
  },
  {
    title: 'Configuration',
    image: imgConfiguration,
    imageLight: imgConfigurationL,
    description:
      'Customize build settings, environment variables, and runtime options for each environment',
    id: 'feature-configuration',
    pos: {
      x: { md: 700 },
      y: { md: 285 },
    },
    arrow: {
      left: '0',
      top: '50%',
    },
  },
  {
    title: 'Status checks',
    image: imgStatusChecks,
    imageLight: imgStatusChecksL,
    description:
      'Post-deployment checks to ensure your application is running as expected before promoting to production',
    id: 'feature-status-checks',
    pos: {
      x: { md: 390 },
      y: { md: 650 },
    },
    arrow: {
      left: '50%',
      top: '100%',
    },
  },
  {
    title: 'Periodic triggers',
    image: imgPeriodicTriggers,
    imageLight: imgPeriodicTriggersL,
    description:
      'Call specified endpoints with scheduled tasks for automated behaviour',
    id: 'feature-triggers',
    pos: {
      x: { md: 425 },
      y: { md: 285 },
    },
    arrow: {
      left: '0',
      top: '50%',
    },
  },
  {
    title: 'Persistent volumes',
    image: imgVolumes,
    imageLight: imgVolumesL,
    description: 'Store and access persistent files across deployments',
    id: 'feature-volumes',
    maxWidth: 500,
    pos: {
      x: { md: 56 },
      y: { md: 236 },
    },
    arrow: {
      left: '50%',
      top: '100%',
    },
  },
  {
    title: 'Snippet injection',
    image: imgSnippets,
    imageLight: imgSnippetsL,
    description: 'Insert custom code snippets into your HTML instantly',
    id: 'feature-snippets',
    maxWidth: 300,
    pos: {
      x: { md: 30 },
      y: { md: 304 },
    },
  },
]
