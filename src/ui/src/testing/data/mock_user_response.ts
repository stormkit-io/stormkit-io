interface UserResponse {
  user: User | null;
  accounts: Array<{ provider: Provider; url: string; displayName: string }>;
  ok: boolean;
}

export default (): UserResponse => ({
  user: {
    displayName: "stormkit",
    avatar: "https://avatars2.githubusercontent.com/u/3321893?v=4",
    email: "foo@bar",
    memberSince: 1551184200,
    id: "1",
    fullName: "Foo Bar",
    package: { id: "free" },
  },
  accounts: [
    { provider: "gitlab", url: "", displayName: "Stormkit" },
    {
      provider: "bitbucket",
      url: "https://bitbucket.org/%7B6e4d532c-e1b6-4496-90cb-f94f09af2bda%7D/",
      displayName: "stormkit",
    },
    {
      provider: "github",
      url: "https://api.github.com/users/stormkit",
      displayName: "stormkit",
    },
  ],
  ok: true,
});
