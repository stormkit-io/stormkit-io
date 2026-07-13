interface Props {
  teamId: string;
}

export default ({ teamId }: Props): TeamMember[] => [
  {
    id: "1",
    teamId,
    userId: "1501",
    role: "owner",
    status: true,
    displayName: "dlorenzo",
    email: "dlorenzo@stormkit.io",
  },
  {
    id: "2",
    teamId,
    userId: "1502",
    role: "admin",
    status: true,
    displayName: "svalerie",
    email: "svalerie@stormkit.io",
  },
  {
    id: "3",
    teamId,
    userId: "1503",
    role: "developer",
    status: true,
    displayName: "msanchez",
    email: "msanchez@stormkit.io",
  },
];
