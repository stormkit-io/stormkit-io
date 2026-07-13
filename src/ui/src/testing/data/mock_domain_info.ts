export default (): DomainLookup => ({
  tls: {
    startDate: 1593500298,
    endDate: 1601276298,
    subject: "app.stormkit.io",
    issuer: {
      country: ["US"],
      organization: ["Let's Encrypt"],
      name: "Let's Encrypt Authority X3",
    },
    dnsNames: ["app.stormkit.io"],
    serialNo: "343486059919871512067800302572875759206296",
    signatureAlgorithm: "SHA256-RSA",
  },
  domainName: "app.stormkit.io",
  dns: {
    verified: true,
    txt: {
      value: "txt-value",
      name: "txt-host",
      lookup: "txt-value.app.stormkit.io",
      records: {},
    },
  },
});
