import Typography from '@mui/material/Typography'
import Box from '@mui/material/Box'
import Link from '@mui/material/Link'
import Header from '~/components/Header'
import Footer from '~/components/Footer'
import StormkitToolImg from '~/assets/images/stormkit-tool.png'
import { useScrollToHash } from '~/helpers/scroll'

export const fetchData: FetchDataFunc = async () => {
  return {
    head: {
      title: 'What makes Stormkit a great Netlify alternative?',
      description:
        'Stormkit is a fully-featured, powerful, and self-hostable alternative to Netlify.',
    },
    context: {},
  }
}

interface Props {
  children: React.ReactNode
  id?: string
}

function Subtitle({ children, id }: Props) {
  return (
    <Typography
      id={id}
      variant="subtitle2"
      sx={{ fontSize: 20, fontWeight: 600 }}
    >
      {children}
    </Typography>
  )
}

const differences = [
  ['Self-Hostability vs. Managed Service', '#hosting-approach'],
  ['Deployment Flexibility', '#flexibility'],
  ['Customizability and Control', '#customizability'],
  ['Pricing Transparency and Predictability', '#pricing'],
  ['Features and Integrations', '#features'],
  ['Security and Compliance', '#security'],
]

export default function Netlify() {
  useScrollToHash()

  return (
    <Box
      sx={{
        minHeight: '100vh',
        display: 'flex',
        flexDirection: 'column',
        bgcolor: 'background.default',
        color: 'primary.contrastText',
      }}
    >
      <Header />
      <Box
        sx={{
          flex: 1,
          m: 'auto',
          bgcolor: 'rgba(0,0,0,0.1)',
          my: 4,
          p: { xs: 2, md: 4 },
        }}
        maxWidth="768px"
      >
        <Typography
          variant="h1"
          sx={{
            fontWeight: 600,
            fontSize: { xs: 24, md: 36 },
            mb: 4,
            textAlign: 'center',
          }}
        >
          What makes Stormkit a
          <br /> great Netlify alternative
        </Typography>
        <Typography>
          When it comes to hosting and deploying frontend applications, both{' '}
          <strong>Stormkit</strong> and <b>Netlify</b> are prominent platforms.
          While Netlify is well-known for its ease of use and cloud-based
          hosting, Stormkit stands out by offering self-hosting capabilities,
          giving developers more control over their infrastructure. In this
          guide, we'll compare the two platforms across key categories to help
          you determine which one best suits your needs.
        </Typography>
        <Box
          component="img"
          src={StormkitToolImg}
          alt="Stormkit - Self-Hosted Frontend Hosting platform"
          sx={{
            mt: { xs: 4, lg: 8 },
            mb: { xs: 4, lg: 8 },
            boxShadow: {
              xs: '0px 0px 15px 0px rgba(255,255,255,0.2)',
              lg: '0px 0px 30px 0px rgba(255,255,255,0.2)',
            },
            borderRadius: '8px',
            position: 'relative',
            width: { xs: '100%', lg: '994px' },
            marginLeft: { xs: 0, lg: '-144px' },
          }}
        ></Box>
        <Box
          sx={{
            border: '1px solid rgba(255,255,255,0.05)',
            bgcolor: 'rgba(0,0,0,0.3)',
            p: 4,
            mb: 8,
          }}
        >
          <Typography component="li" sx={{ listStyle: 'none' }}>
            What are the differences between Stormkit and Netlify?
          </Typography>
          <Box component="ol" sx={{ p: 0, mt: 2, mb: 0 }}>
            {differences.map(([text, hash]) => (
              <Box
                key={hash}
                component="li"
                sx={{ listStylePosition: 'inside' }}
              >
                <Link href={hash} sx={{ color: 'white' }}>
                  {text}
                </Link>
              </Box>
            ))}
          </Box>
        </Box>
        <Subtitle id="hosting-approach">
          1. Self-Hostability vs. Managed Service
        </Subtitle>
        <Typography component="ul" sx={{ my: 2 }}>
          <Typography component="li">
            Stormkit offers the flexibility to either{' '}
            <Link href="/docs/self-hosting/getting-started">
              self-host your applications
            </Link>{' '}
            or use its managed cloud service. For users or companies that
            require complete control over their hosting environment, whether for
            compliance, security, or customization needs, Stormkit provides the
            tools to deploy applications on your own infrastructure.{' '}
            <strong>
              This makes Stormkit ideal for businesses that prefer or need
              on-premise hosting
            </strong>
            .
          </Typography>
          <Typography component="li" sx={{ mt: 2 }}>
            <Link
              sx={{ color: 'white', fontWeight: 'bold' }}
              href="https://netlify.com"
              rel="nofollow noreferrer"
              target="_blank"
            >
              Netlify
            </Link>
            , on the other hand, is a fully cloud-based platform, where all your
            sites are hosted on Netlify's infrastructure. It's an ideal solution
            for users who want to focus solely on development without worrying
            about server management. All deployments are automatically managed
            in their data centers, ensuring a hands-off experience.
          </Typography>
        </Typography>
        <Typography sx={{ mt: 4 }}>
          Watch the following tutorial on how to{' '}
          <Link
            href="https://www.youtube.com/watch?v=bLw0r3VKLrg&feature=youtu.be"
            target="_blank"
            rel="noopener noreferrer"
          >
            install Stormkit on an Ubuntu Server and deploy a Next.js
            Application
          </Link>{' '}
          within a few minutes.
        </Typography>
        <Typography
          sx={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            my: 4,
          }}
        >
          <Box
            component="iframe"
            src="https://www.youtube.com/embed/bLw0r3VKLrg?si=Kyn6iDZLGizbnKoq"
            title="Install Stormkit on Ubuntu Server and Deploy a Next.js App"
            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
            referrerPolicy="strict-origin-when-cross-origin"
            frameBorder="0"
            allowFullScreen
            sx={{
              width: { xs: '100%', md: '560px' },
              height: { xs: '250px', md: '315px' },
            }}
          />
        </Typography>
        <Subtitle id="flexibility">2. Deployment Flexibility</Subtitle>
        <Typography component="ul" sx={{ mt: 2 }}>
          <Typography component="li">
            Stormkit provides automatic deployments via Git integration but
            extends its CI/CD functionality through support for{' '}
            <Link
              href="https://github.com/stormkit-io/runner"
              target="_blank"
              rel="noopener noreferrer"
            >
              GitHub Actions
            </Link>{' '}
            when self-hosting. This enables advanced CI workflows and greater
            flexibility for handling complex deployment scenarios. Developers
            can set up custom workflows that allow for more control, ideal for
            larger teams or projects with complex requirements.
          </Typography>
          <Typography component="li" sx={{ my: 2 }}>
            Netlify integrates with popular Git providers like GitHub, GitLab,
            and Bitbucket. It automatically deploys sites every time you push
            changes to your repository, making it an easy choice for projects
            that need fast, frequent updates. Each push creates a deploy
            preview, enabling you to view and test changes in a separate
            environment before going live.
          </Typography>
        </Typography>
        <Subtitle id="customizability">3. Customizability and Control</Subtitle>
        <Typography component="ul" sx={{ mt: 2 }}>
          <Typography component="li">
            Stormkit's key advantage is its <strong>self-hosting</strong>{' '}
            option. This gives developers the ability to host applications on
            their own infrastructure, offering more control over security,
            performance, and scalability. This makes Stormkit the ideal solution
            for companies with stricter data governance policies or those
            looking to customize their infrastructure setup.
          </Typography>
          <Typography component="li" sx={{ my: 2 }}>
            Netlify is a fully managed platform, meaning you don't have direct
            access to the underlying servers. For many users, this simplicity is
            beneficial as it allows them to focus solely on development without
            managing infrastructure. However, it also limits users who need more
            granular control over their environment.
          </Typography>
        </Typography>
        <Subtitle id="pricing">
          4. Pricing Transparency and Predictability
        </Subtitle>
        <Typography component="ul" sx={{ mt: 2 }}>
          <Typography component="li">
            Stormkit's Cloud offering starts from 20 USD per month, whereas
            self-hosted users can start with a free tier. In both cases, the
            pricing is flexible and becomes more flexible with the ability to
            self-host. This can lead to more predictable costs, especially for
            larger applications or enterprises that want to avoid paying for
            excess resources. You can{' '}
            <Link href="/#pricing">check our pricing here.</Link>
          </Typography>
          <Typography component="li" sx={{ my: 2 }}>
            Netlify offers a generous free tier that includes automated
            Git-based deployments, a global CDN, and basic form handling. As
            your project grows, you can scale to paid tiers, which offer
            additional features like team collaboration, priority support, and
            more.
          </Typography>
        </Typography>
        <Subtitle id="features">4. Features and Integrations</Subtitle>
        <Typography component="ul" sx={{ mt: 2 }}>
          <Typography component="li">
            Stormkit goes beyond basic hosting by offering features like{' '}
            <Link href="/docs/features/snippets">
              instant snippet injection
            </Link>
            , <Link href="/docs/deployments/status-checks">status checks</Link>,{' '}
            <Link href="/docs/features/writing-api">serverless functions</Link>,{' '}
            <Link href="/docs/features/analytics">analytics</Link> and more.
          </Typography>
          <Typography component="li" sx={{ my: 2 }}>
            Netlify provides serverless functions (Netlify Functions), enabling
            backend functionality like form handling and user authentication
            directly on the platform. Additionally, Netlify offers an extensive
            plugin system that allows developers to extend their sites with
            features such as image optimization, SEO enhancements, and more.
          </Typography>
        </Typography>
        <Subtitle id="security">5. Security and Compliance</Subtitle>
        <Typography component="ul" sx={{ mt: 2 }}>
          <Typography component="li">
            For businesses requiring enhanced security, Stormkit's self-hosting
            option allows you to implement your own security measures and comply
            with regional laws such as GDPR or other data protection
            regulations. This makes Stormkit a more attractive option for
            industries with high-security needs.
          </Typography>
          <Typography component="li" sx={{ my: 2 }}>
            Netlify provides built-in security features, including automatic
            SSL/TLS certificates for all sites, meaning every deployment is
            secured by default. However, as a fully managed platform, you're
            dependent on Netlify's security protocols, which may not meet more
            stringent compliance requirements.
          </Typography>
        </Typography>
        <Subtitle id="conclusion">Conclusion</Subtitle>
        <Typography component="ul" sx={{ mt: 2 }}>
          <Typography component="li">
            In summary, Netlify is a great option for simplicity and ease of use
            for cloud-based hosting, while Stormkit stands out with its
            self-hosting capabilities and greater control, making it ideal for
            advanced use cases and teams with specific infrastructure
            requirements.
          </Typography>
          <Typography component="li" sx={{ my: 2 }}>
            Ultimately, the choice between Stormkit and Netlify depends on your
            project's needs. If you want maximum control over your hosting
            environment, Stormkit is the way to go. If you prefer a fully
            managed solution with minimal setup, Netlify might be a better fit.
          </Typography>
        </Typography>
        <Subtitle id="whats-next">What's Next?</Subtitle>
        <Typography sx={{ mt: 2 }}>
          Check our{' '}
          <Link href="/docs/self-hosting/getting-started">documentation</Link>{' '}
          to get started with self-hosting.
        </Typography>
      </Box>
      <Footer maxWidth="lg" />
    </Box>
  )
}
