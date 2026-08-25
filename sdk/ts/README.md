# @glueops/apiary-sdk@dev

A TypeScript SDK client for the 127.0.0.1 API.

## Usage

First, install the SDK from npm.

```bash
npm install @glueops/apiary-sdk --save
```

Next, try it out.


```ts
import {
  Configuration,
  APIKeysApi,
} from '@glueops/apiary-sdk';
import type { ApiKeysCreateRequest } from '@glueops/apiary-sdk';

async function example() {
  console.log("🚀 Testing @glueops/apiary-sdk SDK...");
  const config = new Configuration({ 
    // To configure API key authorization: OrgKeySecret
    apiKey: "YOUR API KEY",
    // To configure API key authorization: OrgKeyAuth
    apiKey: "YOUR API KEY",
    // To configure API key authorization: ApiKeyAuth
    apiKey: "YOUR API KEY",
    // Configure HTTP bearer authorization: BearerAuth
    accessToken: "YOUR BEARER TOKEN",
  });
  const api = new APIKeysApi(config);

  const body = {
    // MintAPIKeyRequestBody
    mintAPIKeyRequestBody: ...,
  } satisfies ApiKeysCreateRequest;

  try {
    const data = await api.apiKeysCreate(body);
    console.log(data);
  } catch (error) {
    console.error(error);
  }
}

// Run the test
example().catch(console.error);
```


## Documentation

### API Endpoints

All URIs are relative to *http://127.0.0.1:8080*

| Class | Method | HTTP request | Description
| ----- | ------ | ------------ | -------------
*APIKeysApi* | [**apiKeysCreate**](docs/APIKeysApi.md#apikeyscreate) | **POST** /api/v1/api-keys | Mint an API key
*APIKeysApi* | [**apiKeysList**](docs/APIKeysApi.md#apikeyslist) | **GET** /api/v1/api-keys | List API keys
*APIKeysApi* | [**apiKeysRevoke**](docs/APIKeysApi.md#apikeysrevoke) | **DELETE** /api/v1/api-keys/{id} | Revoke an API key
*AdminApi* | [**adminImpersonateEnd**](docs/AdminApi.md#adminimpersonateend) | **POST** /api/v1/admin/impersonate/end | End the current impersonated session
*AdminApi* | [**adminImpersonateStart**](docs/AdminApi.md#adminimpersonatestart) | **POST** /api/v1/admin/impersonate | Begin impersonating a user
*AuthApi* | [**authForgotPassword**](docs/AuthApi.md#authforgotpassword) | **POST** /api/v1/auth/forgot-password | Request a password reset link
*AuthApi* | [**authLogin**](docs/AuthApi.md#authlogin) | **POST** /api/v1/auth/login | Authenticate a user
*AuthApi* | [**authLogout**](docs/AuthApi.md#authlogout) | **POST** /api/v1/auth/logout | Logout current session
*AuthApi* | [**authMfaVerify**](docs/AuthApi.md#authmfaverify) | **POST** /api/v1/auth/mfa/verify | Complete MFA challenge
*AuthApi* | [**authPasskeyLoginBegin**](docs/AuthApi.md#authpasskeyloginbegin) | **POST** /api/v1/auth/passkey/login/begin | Begin passkey login
*AuthApi* | [**authPasskeyLoginFinish**](docs/AuthApi.md#authpasskeyloginfinish) | **POST** /api/v1/auth/passkey/login/finish | Complete passkey login
*AuthApi* | [**authPasskeyRegisterBegin**](docs/AuthApi.md#authpasskeyregisterbegin) | **POST** /api/v1/auth/passkey/register/begin | Begin passkey registration
*AuthApi* | [**authPasskeyRegisterFinish**](docs/AuthApi.md#authpasskeyregisterfinish) | **POST** /api/v1/auth/passkey/register/finish | Complete passkey registration
*AuthApi* | [**authPasskeysDelete**](docs/AuthApi.md#authpasskeysdelete) | **DELETE** /api/v1/auth/passkeys/{id} | Remove a passkey
*AuthApi* | [**authPasskeysList**](docs/AuthApi.md#authpasskeyslist) | **GET** /api/v1/auth/passkeys | List registered passkeys
*AuthApi* | [**authRefresh**](docs/AuthApi.md#authrefresh) | **POST** /api/v1/auth/refresh | Refresh auth tokens
*AuthApi* | [**authRegister**](docs/AuthApi.md#authregister) | **POST** /api/v1/auth/register | Create an account and its first organization
*AuthApi* | [**authResendVerification**](docs/AuthApi.md#authresendverification) | **POST** /api/v1/auth/resend-verification | Send a new confirmation link
*AuthApi* | [**authResetPassword**](docs/AuthApi.md#authresetpassword) | **POST** /api/v1/auth/reset-password | Set a new password using a reset token
*AuthApi* | [**authSocialCallback**](docs/AuthApi.md#authsocialcallback) | **GET** /api/v1/auth/{provider}/callback | Handle social login callback
*AuthApi* | [**authSocialStart**](docs/AuthApi.md#authsocialstart) | **POST** /api/v1/auth/{provider}/start | Begin social login
*AuthApi* | [**authTotpConfirm**](docs/AuthApi.md#authtotpconfirm) | **POST** /api/v1/auth/totp/verify | Confirm TOTP setup
*AuthApi* | [**authTotpDisable**](docs/AuthApi.md#authtotpdisable) | **DELETE** /api/v1/auth/totp | Disable TOTP
*AuthApi* | [**authTotpSetup**](docs/AuthApi.md#authtotpsetup) | **POST** /api/v1/auth/totp/setup | Start TOTP setup
*AuthApi* | [**authVerifyEmail**](docs/AuthApi.md#authverifyemail) | **POST** /api/v1/auth/verify-email | Confirm an email address
*EstatesApi* | [**estatesCreate**](docs/EstatesApi.md#estatescreate) | **POST** /api/v1/estates | Register an estate and its credential
*EstatesApi* | [**estatesDelete**](docs/EstatesApi.md#estatesdelete) | **DELETE** /api/v1/estates/{id} | Delete an estate and its credential
*EstatesApi* | [**estatesGet**](docs/EstatesApi.md#estatesget) | **GET** /api/v1/estates/{id} | Get an estate
*EstatesApi* | [**estatesList**](docs/EstatesApi.md#estateslist) | **GET** /api/v1/estates | List estates
*EstatesApi* | [**estatesRotateCredential**](docs/EstatesApi.md#estatesrotatecredential) | **POST** /api/v1/estates/{id}/rotate | Rotate an estate\&#39;s credential
*FirewallApi* | [**firewallBindingsCreate**](docs/FirewallApi.md#firewallbindingscreate) | **POST** /api/v1/firewall/bindings | Bind a ruleset to a fleet, or to the whole organization
*FirewallApi* | [**firewallBindingsDelete**](docs/FirewallApi.md#firewallbindingsdelete) | **DELETE** /api/v1/firewall/bindings/{id} | Remove a binding
*FirewallApi* | [**firewallBindingsList**](docs/FirewallApi.md#firewallbindingslist) | **GET** /api/v1/firewall/bindings | List ruleset bindings
*FirewallApi* | [**firewallBindingsUpdate**](docs/FirewallApi.md#firewallbindingsupdate) | **PUT** /api/v1/firewall/bindings/{id} | Reprioritise or disable a binding
*FirewallApi* | [**firewallPreview**](docs/FirewallApi.md#firewallpreview) | **GET** /api/v1/firewall/preview | Preview the rules a fleet would receive
*FirewallApi* | [**firewallRulesCreate**](docs/FirewallApi.md#firewallrulescreate) | **POST** /api/v1/firewall/rulesets/{id}/rules | Add a rule to a ruleset
*FirewallApi* | [**firewallRulesDelete**](docs/FirewallApi.md#firewallrulesdelete) | **DELETE** /api/v1/firewall/rules/{id} | Delete a rule
*FirewallApi* | [**firewallRulesList**](docs/FirewallApi.md#firewallruleslist) | **GET** /api/v1/firewall/rulesets/{id}/rules | List a ruleset\&#39;s rules
*FirewallApi* | [**firewallRulesUpdate**](docs/FirewallApi.md#firewallrulesupdate) | **PUT** /api/v1/firewall/rules/{id} | Replace a rule
*FirewallApi* | [**firewallRulesetsCreate**](docs/FirewallApi.md#firewallrulesetscreate) | **POST** /api/v1/firewall/rulesets | Create a firewall ruleset
*FirewallApi* | [**firewallRulesetsDelete**](docs/FirewallApi.md#firewallrulesetsdelete) | **DELETE** /api/v1/firewall/rulesets/{id} | Delete a firewall ruleset
*FirewallApi* | [**firewallRulesetsList**](docs/FirewallApi.md#firewallrulesetslist) | **GET** /api/v1/firewall/rulesets | List firewall rulesets
*FirewallApi* | [**firewallRulesetsUpdate**](docs/FirewallApi.md#firewallrulesetsupdate) | **PUT** /api/v1/firewall/rulesets/{id} | Rename a firewall ruleset
*FleetApi* | [**fleetBeeAppendLogs**](docs/FleetApi.md#fleetbeeappendlogs) | **POST** /api/v1/agent/tasks/{id}/logs | Stream task output back to the control plane
*FleetApi* | [**fleetBeeEnrol**](docs/FleetApi.md#fleetbeeenrol) | **POST** /api/v1/agent/enrol | Enrol a fleet agent into an organization
*FleetApi* | [**fleetBeeHeartbeat**](docs/FleetApi.md#fleetbeeheartbeat) | **POST** /api/v1/agent/heartbeat | Report liveness and collect work
*FleetApi* | [**fleetBeeReportTask**](docs/FleetApi.md#fleetbeereporttask) | **POST** /api/v1/agent/tasks/{id}/status | Report progress or the outcome of a task
*FleetApi* | [**fleetBeesDelete**](docs/FleetApi.md#fleetbeesdelete) | **DELETE** /api/v1/fleet/bees/{id} | Remove a fleet agent
*FleetApi* | [**fleetBeesList**](docs/FleetApi.md#fleetbeeslist) | **GET** /api/v1/fleet/bees | List fleet agents
*FleetApi* | [**fleetEnrolmentTokensMint**](docs/FleetApi.md#fleetenrolmenttokensmint) | **POST** /api/v1/fleet/enrolment-tokens | Mint an enrolment token
*FleetApi* | [**fleetHivesCreate**](docs/FleetApi.md#fleethivescreate) | **POST** /api/v1/fleet/hives | Create a hive
*FleetApi* | [**fleetHivesDelete**](docs/FleetApi.md#fleethivesdelete) | **DELETE** /api/v1/fleet/hives/{id} | Delete a hive
*FleetApi* | [**fleetHivesList**](docs/FleetApi.md#fleethiveslist) | **GET** /api/v1/fleet/hives | List hives
*FleetApi* | [**fleetHivesSync**](docs/FleetApi.md#fleethivessync) | **POST** /api/v1/fleet/hives/{id}/sync | Refresh a hive\&#39;s capacity observations now
*FleetApi* | [**fleetHivesUpdate**](docs/FleetApi.md#fleethivesupdate) | **PATCH** /api/v1/fleet/hives/{id} | Update a hive\&#39;s configuration
*FleetApi* | [**fleetHostsArrive**](docs/FleetApi.md#fleethostsarrive) | **POST** /api/v1/fleet/fleetless/{hypervisor_id}/arrive | Add a standalone machine to a hive
*FleetApi* | [**fleetHostsFleetless**](docs/FleetApi.md#fleethostsfleetless) | **GET** /api/v1/fleet/fleetless | List machines belonging to no hive
*FleetApi* | [**fleetHypervisorsCordon**](docs/FleetApi.md#fleethypervisorscordon) | **POST** /api/v1/fleet/hypervisors/{id}/cordon | Cordon or uncordon a hypervisor
*FleetApi* | [**fleetHypervisorsDepart**](docs/FleetApi.md#fleethypervisorsdepart) | **POST** /api/v1/fleet/hypervisors/{id}/depart | Remove a hypervisor from its hive
*FleetApi* | [**fleetHypervisorsGet**](docs/FleetApi.md#fleethypervisorsget) | **GET** /api/v1/fleet/hypervisors/{id} | Get one hypervisor
*FleetApi* | [**fleetHypervisorsList**](docs/FleetApi.md#fleethypervisorslist) | **GET** /api/v1/fleet/hypervisors | List machines with their three status axes
*FleetApi* | [**fleetHypervisorsReprovision**](docs/FleetApi.md#fleethypervisorsreprovision) | **POST** /api/v1/fleet/hypervisors/{id}/reprovision | Redrive a machine\&#39;s onboarding
*FleetApi* | [**fleetTasksCreate**](docs/FleetApi.md#fleettaskscreate) | **POST** /api/v1/fleet/tasks | Queue work for a bee
*FleetApi* | [**fleetTasksLogs**](docs/FleetApi.md#fleettaskslogs) | **GET** /api/v1/fleet/tasks/{id}/logs | Read a task\&#39;s streamed output
*FleetApi* | [**sshKeysList**](docs/FleetApi.md#sshkeyslist) | **GET** /api/v1/ssh-keys | List fleet SSH keys
*FleetApi* | [**sshKeysMint**](docs/FleetApi.md#sshkeysmint) | **POST** /api/v1/ssh-keys | Mint a fleet SSH key
*FleetApi* | [**sshKeysRevoke**](docs/FleetApi.md#sshkeysrevoke) | **DELETE** /api/v1/ssh-keys/{id} | Revoke a fleet SSH key
*InvitationsApi* | [**invitationsAccept**](docs/InvitationsApi.md#invitationsaccept) | **POST** /api/v1/invitations/accept | Accept an invitation
*InvitationsApi* | [**invitationsPreview**](docs/InvitationsApi.md#invitationspreview) | **GET** /api/v1/invitations/preview | Describe an invitation
*InvitationsApi* | [**orgInvitationsCreate**](docs/InvitationsApi.md#orginvitationscreate) | **POST** /api/v1/invitations | Invite someone into this organization
*InvitationsApi* | [**orgInvitationsList**](docs/InvitationsApi.md#orginvitationslist) | **GET** /api/v1/invitations | List this organization\&#39;s invitations
*InvitationsApi* | [**orgInvitationsRevoke**](docs/InvitationsApi.md#orginvitationsrevoke) | **DELETE** /api/v1/invitations/{id} | Withdraw an invitation
*MeApi* | [**meDeleteAccount**](docs/MeApi.md#medeleteaccount) | **DELETE** /api/v1/me | Delete your account
*MeApi* | [**meSoleOwnerships**](docs/MeApi.md#mesoleownerships) | **GET** /api/v1/me/sole-ownerships | Organizations this account solely owns
*OrganizationApi* | [**orgMembersList**](docs/OrganizationApi.md#orgmemberslist) | **GET** /api/v1/organization/members | List the members of this organization
*OrganizationApi* | [**orgMembersRemove**](docs/OrganizationApi.md#orgmembersremove) | **DELETE** /api/v1/organization/members/{id} | Remove a member from this organization
*OrganizationApi* | [**orgMembersUpdateRole**](docs/OrganizationApi.md#orgmembersupdaterole) | **PATCH** /api/v1/organization/members/{id} | Change a member\&#39;s role
*OrganizationApi* | [**orgPoliciesClear**](docs/OrganizationApi.md#orgpoliciesclear) | **DELETE** /api/v1/organization/policies/{kind} | Return a policy to inherit
*OrganizationApi* | [**orgPoliciesImpact**](docs/OrganizationApi.md#orgpoliciesimpact) | **GET** /api/v1/organization/policies/{kind}/impact | How many members this policy would lock out
*OrganizationApi* | [**orgPoliciesList**](docs/OrganizationApi.md#orgpolicieslist) | **GET** /api/v1/organization/policies | List this organization\&#39;s policies
*OrganizationApi* | [**orgPoliciesSet**](docs/OrganizationApi.md#orgpoliciesset) | **PUT** /api/v1/organization/policies/{kind} | Set a policy for this organization
*OrganizationsApi* | [**orgDelete**](docs/OrganizationsApi.md#orgdelete) | **DELETE** /api/v1/orgs/current | Delete this organization
*PlatformApi* | [**platformAccessRulesCreate**](docs/PlatformApi.md#platformaccessrulescreate) | **POST** /api/v1/platform/access-rules | Create an auth access rule
*PlatformApi* | [**platformAccessRulesDelete**](docs/PlatformApi.md#platformaccessrulesdelete) | **DELETE** /api/v1/platform/access-rules/{id} | Delete an auth access rule
*PlatformApi* | [**platformAccessRulesList**](docs/PlatformApi.md#platformaccessruleslist) | **GET** /api/v1/platform/access-rules | List auth access rules
*PlatformApi* | [**platformAccessRulesUpdate**](docs/PlatformApi.md#platformaccessrulesupdate) | **PUT** /api/v1/platform/access-rules/{id} | Change a rule\&#39;s policy
*PlatformApi* | [**platformPoliciesClear**](docs/PlatformApi.md#platformpoliciesclear) | **DELETE** /api/v1/platform/policies/{kind} | Return a platform policy to inherit
*PlatformApi* | [**platformPoliciesImpact**](docs/PlatformApi.md#platformpoliciesimpact) | **GET** /api/v1/platform/policies/{kind}/impact | How many users this policy would lock out
*PlatformApi* | [**platformPoliciesList**](docs/PlatformApi.md#platformpolicieslist) | **GET** /api/v1/platform/policies | List platform policies
*PlatformApi* | [**platformPoliciesSet**](docs/PlatformApi.md#platformpoliciesset) | **PUT** /api/v1/platform/policies/{kind} | Set a platform policy
*ProfileApi* | [**meChangePassword**](docs/ProfileApi.md#mechangepassword) | **PUT** /api/v1/me/password | Change password
*ProfileApi* | [**meGet**](docs/ProfileApi.md#meget) | **GET** /api/v1/me | Get current user profile
*ProfileApi* | [**meOrgsList**](docs/ProfileApi.md#meorgslist) | **GET** /api/v1/me/orgs | List my organizations
*ProfileApi* | [**meUpdate**](docs/ProfileApi.md#meupdate) | **PATCH** /api/v1/me | Update current user profile


### Models

- [APIKeyView](docs/APIKeyView.md)
- [AcceptRequestBody](docs/AcceptRequestBody.md)
- [AcceptResponseBody](docs/AcceptResponseBody.md)
- [AccessRuleView](docs/AccessRuleView.md)
- [ArriveRequestBody](docs/ArriveRequestBody.md)
- [ArriveResponseBody](docs/ArriveResponseBody.md)
- [AuthenticatorSelection](docs/AuthenticatorSelection.md)
- [BeeView](docs/BeeView.md)
- [BindingView](docs/BindingView.md)
- [CapacityView](docs/CapacityView.md)
- [ChangePasswordRequestBody](docs/ChangePasswordRequestBody.md)
- [ChangePasswordResponseBody](docs/ChangePasswordResponseBody.md)
- [CordonRequestBody](docs/CordonRequestBody.md)
- [CorosyncLinkInput](docs/CorosyncLinkInput.md)
- [CreateBindingRequestBody](docs/CreateBindingRequestBody.md)
- [CreateHiveRequestBody](docs/CreateHiveRequestBody.md)
- [CreateRequestBody](docs/CreateRequestBody.md)
- [CreateRequestBody1](docs/CreateRequestBody1.md)
- [CreateRequestBody2](docs/CreateRequestBody2.md)
- [CreateRulesetRequestBody](docs/CreateRulesetRequestBody.md)
- [CreateTaskRequestBody](docs/CreateTaskRequestBody.md)
- [CreateTaskResponseBody](docs/CreateTaskResponseBody.md)
- [CredentialAssertion](docs/CredentialAssertion.md)
- [CredentialCreation](docs/CredentialCreation.md)
- [CredentialDescriptor](docs/CredentialDescriptor.md)
- [CredentialParameter](docs/CredentialParameter.md)
- [DeleteAccountRequestBody](docs/DeleteAccountRequestBody.md)
- [DeleteAccountResponseBody](docs/DeleteAccountResponseBody.md)
- [DeleteBeeResponseBody](docs/DeleteBeeResponseBody.md)
- [DeleteOrgRequestBody](docs/DeleteOrgRequestBody.md)
- [DeleteOrgResponseBody](docs/DeleteOrgResponseBody.md)
- [DeleteResponseBody](docs/DeleteResponseBody.md)
- [DeleteResponseBody1](docs/DeleteResponseBody1.md)
- [DeleteResponseBody2](docs/DeleteResponseBody2.md)
- [DeletedResponseBody](docs/DeletedResponseBody.md)
- [DepartResponseBody](docs/DepartResponseBody.md)
- [EnrolRequestBody](docs/EnrolRequestBody.md)
- [EnrolResponseBody](docs/EnrolResponseBody.md)
- [ErrorDetail](docs/ErrorDetail.md)
- [ErrorModel](docs/ErrorModel.md)
- [EstateView](docs/EstateView.md)
- [FleetlessHostView](docs/FleetlessHostView.md)
- [ForgotPasswordRequestBody](docs/ForgotPasswordRequestBody.md)
- [ForgotPasswordResponseBody](docs/ForgotPasswordResponseBody.md)
- [GetMeResponseBody](docs/GetMeResponseBody.md)
- [GetMyOrganizationsResponseBody](docs/GetMyOrganizationsResponseBody.md)
- [HeartbeatRequestBody](docs/HeartbeatRequestBody.md)
- [HeartbeatResponseBody](docs/HeartbeatResponseBody.md)
- [HiveDetailView](docs/HiveDetailView.md)
- [HiveView](docs/HiveView.md)
- [HypervisorView](docs/HypervisorView.md)
- [ImpactResponseBody](docs/ImpactResponseBody.md)
- [ImpersonateEndRequestBody](docs/ImpersonateEndRequestBody.md)
- [ImpersonateEndResponseBody](docs/ImpersonateEndResponseBody.md)
- [ImpersonateStartRequestBody](docs/ImpersonateStartRequestBody.md)
- [ImpersonateStartResponseBody](docs/ImpersonateStartResponseBody.md)
- [InvitationView](docs/InvitationView.md)
- [Item](docs/Item.md)
- [KeyView](docs/KeyView.md)
- [ListAPIKeysResponseBody](docs/ListAPIKeysResponseBody.md)
- [ListBeesResponseBody](docs/ListBeesResponseBody.md)
- [ListBindingsResponseBody](docs/ListBindingsResponseBody.md)
- [ListFleetlessResponseBody](docs/ListFleetlessResponseBody.md)
- [ListHivesResponseBody](docs/ListHivesResponseBody.md)
- [ListHypervisorsResponseBody](docs/ListHypervisorsResponseBody.md)
- [ListMembersResponseBody](docs/ListMembersResponseBody.md)
- [ListResponseBody](docs/ListResponseBody.md)
- [ListResponseBody1](docs/ListResponseBody1.md)
- [ListResponseBody2](docs/ListResponseBody2.md)
- [ListResponseBody3](docs/ListResponseBody3.md)
- [ListResponseBody4](docs/ListResponseBody4.md)
- [ListRulesResponseBody](docs/ListRulesResponseBody.md)
- [ListRulesetsResponseBody](docs/ListRulesetsResponseBody.md)
- [LoginRequestBody](docs/LoginRequestBody.md)
- [LoginResponseBody](docs/LoginResponseBody.md)
- [LogoutResponseBody](docs/LogoutResponseBody.md)
- [LogsRequestBody](docs/LogsRequestBody.md)
- [LogsResponseBody](docs/LogsResponseBody.md)
- [MFAStruct](docs/MFAStruct.md)
- [MFAVerifyRequestBody](docs/MFAVerifyRequestBody.md)
- [MFAVerifyResponseBody](docs/MFAVerifyResponseBody.md)
- [MemberView](docs/MemberView.md)
- [MintAPIKeyRequestBody](docs/MintAPIKeyRequestBody.md)
- [MintAPIKeyResponseBody](docs/MintAPIKeyResponseBody.md)
- [MintRequestBody](docs/MintRequestBody.md)
- [MintTokenRequestBody](docs/MintTokenRequestBody.md)
- [MintTokenResponseBody](docs/MintTokenResponseBody.md)
- [MyOrganizationView](docs/MyOrganizationView.md)
- [OrganizationStruct](docs/OrganizationStruct.md)
- [PasskeyBeginLoginResponseBody](docs/PasskeyBeginLoginResponseBody.md)
- [PasskeyBeginRegisterResponseBody](docs/PasskeyBeginRegisterResponseBody.md)
- [PasskeyDeleteResponseBody](docs/PasskeyDeleteResponseBody.md)
- [PasskeyEntry](docs/PasskeyEntry.md)
- [PasskeyFinishLoginRequestBody](docs/PasskeyFinishLoginRequestBody.md)
- [PasskeyFinishLoginResponseBody](docs/PasskeyFinishLoginResponseBody.md)
- [PasskeyFinishRegisterRequestBody](docs/PasskeyFinishRegisterRequestBody.md)
- [PasskeyFinishRegisterResponseBody](docs/PasskeyFinishRegisterResponseBody.md)
- [PasskeyListResponseBody](docs/PasskeyListResponseBody.md)
- [PolicyConfigView](docs/PolicyConfigView.md)
- [PolicyView](docs/PolicyView.md)
- [PreviewResponseBody](docs/PreviewResponseBody.md)
- [PreviewResponseBody1](docs/PreviewResponseBody1.md)
- [PublicKeyCredentialCreationOptions](docs/PublicKeyCredentialCreationOptions.md)
- [PublicKeyCredentialRequestOptions](docs/PublicKeyCredentialRequestOptions.md)
- [RefreshRequestBody](docs/RefreshRequestBody.md)
- [RefreshResponseBody](docs/RefreshResponseBody.md)
- [RefusalView](docs/RefusalView.md)
- [RegisterRequestBody](docs/RegisterRequestBody.md)
- [RegisterResponseBody](docs/RegisterResponseBody.md)
- [RelyingPartyEntity](docs/RelyingPartyEntity.md)
- [RemoveMemberResponseBody](docs/RemoveMemberResponseBody.md)
- [ReportRequestBody](docs/ReportRequestBody.md)
- [ReportResponseBody](docs/ReportResponseBody.md)
- [ResendVerificationRequestBody](docs/ResendVerificationRequestBody.md)
- [ResendVerificationResponseBody](docs/ResendVerificationResponseBody.md)
- [ResetPasswordRequestBody](docs/ResetPasswordRequestBody.md)
- [ResetPasswordResponseBody](docs/ResetPasswordResponseBody.md)
- [RevokeAPIKeyResponseBody](docs/RevokeAPIKeyResponseBody.md)
- [RevokeResponseBody](docs/RevokeResponseBody.md)
- [RevokeResponseBody1](docs/RevokeResponseBody1.md)
- [RotateRequestBody](docs/RotateRequestBody.md)
- [RuleBody](docs/RuleBody.md)
- [RuleView](docs/RuleView.md)
- [RulesetView](docs/RulesetView.md)
- [SetRequestBody](docs/SetRequestBody.md)
- [SoleOwnershipsResponseBody](docs/SoleOwnershipsResponseBody.md)
- [StartAuthResponseBody](docs/StartAuthResponseBody.md)
- [StoredTaskLogLine](docs/StoredTaskLogLine.md)
- [SyncHiveResponseBody](docs/SyncHiveResponseBody.md)
- [TOTPDisableRequestBody](docs/TOTPDisableRequestBody.md)
- [TOTPDisableResponseBody](docs/TOTPDisableResponseBody.md)
- [TOTPSetupResponseBody](docs/TOTPSetupResponseBody.md)
- [TOTPVerifyRequestBody](docs/TOTPVerifyRequestBody.md)
- [TOTPVerifyResponseBody](docs/TOTPVerifyResponseBody.md)
- [TaskLogLine](docs/TaskLogLine.md)
- [TaskLogsResponseBody](docs/TaskLogsResponseBody.md)
- [TaskView](docs/TaskView.md)
- [UpdateBindingRequestBody](docs/UpdateBindingRequestBody.md)
- [UpdateHiveRequestBody](docs/UpdateHiveRequestBody.md)
- [UpdateMeRequestBody](docs/UpdateMeRequestBody.md)
- [UpdateMeResponseBody](docs/UpdateMeResponseBody.md)
- [UpdateRequestBody](docs/UpdateRequestBody.md)
- [UpdateRoleRequestBody](docs/UpdateRoleRequestBody.md)
- [UpdateRoleResponseBody](docs/UpdateRoleResponseBody.md)
- [UpdateRulesetRequestBody](docs/UpdateRulesetRequestBody.md)
- [UserEntity](docs/UserEntity.md)
- [VerifyEmailRequestBody](docs/VerifyEmailRequestBody.md)
- [VerifyEmailResponseBody](docs/VerifyEmailResponseBody.md)

### Authorization


Authentication schemes defined for the API:
<a id="AgentKeyAuth"></a>
#### AgentKeyAuth


- **Type**: API key
- **API key parameter name**: `X-AGENT-Key`
- **Location**: HTTP header
<a id="AgentKeySecret"></a>
#### AgentKeySecret


- **Type**: API key
- **API key parameter name**: `X-AGENT-Secret`
- **Location**: HTTP header
<a id="ApiKeyAuth"></a>
#### ApiKeyAuth


- **Type**: API key
- **API key parameter name**: `X-API-Key`
- **Location**: HTTP header
<a id="BearerAuth"></a>
#### BearerAuth


- **Type**: HTTP Bearer Token authentication (JWT)
<a id="OrgKeyAuth"></a>
#### OrgKeyAuth


- **Type**: API key
- **API key parameter name**: `X-ORG-Key`
- **Location**: HTTP header
<a id="OrgKeySecret"></a>
#### OrgKeySecret


- **Type**: API key
- **API key parameter name**: `X-ORG-Secret`
- **Location**: HTTP header

## About

This TypeScript SDK client supports the [Fetch API](https://fetch.spec.whatwg.org/)
and is automatically generated by the
[OpenAPI Generator](https://openapi-generator.tech) project:

- API version: `dev`
- Package version: `dev`
- Generator version: `7.24.0`
- Build package: `org.openapitools.codegen.languages.TypeScriptFetchClientCodegen`

The generated npm module supports the following:

- Environments
  * Node.js
  * Webpack
  * Browserify
- Language levels
  * ES5 - you must have a Promises/A+ library installed
  * ES6
- Module systems
  * CommonJS
  * ES6 module system


## Development

### Building

To build the TypeScript source code, you need to have Node.js and npm installed.
After cloning the repository, navigate to the project directory and run:

```bash
npm install
npm run build
```

### Publishing

Once you've built the package, you can publish it to npm:

```bash
npm publish
```

## License

[]()
