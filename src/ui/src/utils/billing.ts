const isDev = process.env.NODE_ENV === "development";

export const portalLink = isDev
  ? "https://billing.stripe.com/p/login/test_4gw9CvdOF3eabhSeUU"
  : "https://billing.stripe.com/p/login/9AQ7sKfcx2Or41ibII";

export const paymentLink = {
  premium: "https://buy.stripe.com/7sY3cwebC1TEesO8qXbAs06",
  ultimate: "https://buy.stripe.com/eVacOwbDc3dW2IgdQU",
};

export const subscriptionLink = (packageId?: string) =>
  packageId === "free" || !packageId ? paymentLink.premium : portalLink;
