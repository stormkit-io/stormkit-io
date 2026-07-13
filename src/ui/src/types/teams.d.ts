declare type TeamRole = "owner" | "admin" | "developer";

declare interface Team {
  id: string;
  name: string;
  slug: string;
  isDefault: boolean;
  currentUserRole: TeamRole;
}

declare interface TeamMember {
  id: string;
  userId: string;
  teamId: string;
  firstName?: string;
  lastName?: string;
  fullName?: string;
  avatar?: string;
  displayName: string;
  email: string;
  role: TeamRole;
  status: boolean;
}
