JWT time-of-day login windows are broken.

A user limited to 11:00–22:00 used to be accepted at 13:00, with a remaining window of nine hours. A user limited to 22:00–06:00 used to be accepted at 23:30, with six and a half hours left until 06:00. Users with no time restrictions used to always be allowed.

Now every login with a times-of-day restriction is rejected, including those two cases, and users with an empty restriction list are rejected as well.
