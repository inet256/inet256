package inet256_test

// func TestNode(t *testing.T) {
// 	swarmtest.TestSuiteSwarm(t, func(t testing.TB, n int) []p2p.Swarm {
// 		xs := make([]p2p.Swarm)
// 		ps := make(inet256.PeerStore, len(xs))
// 		taddrs := make([]string, len(xs))
// 		for i := range xs {
// 			usw, err := udpswarm.New("127.0.0.1")
// 			require.NoError(err)
// 			ssw := mocksecswarm.New(usw)
// 			taddrs := make([]string,)
// 			pk := p2ptest.NewTestKey(t, n)
// 			ps[i] = NewPeerStore()
// 			x[i] = inet256.NewNode(inet256.Params{
// 				Networks: []inet256.NetworkSpec{
// 					{Name: "onehop", Factory: inet256.OneHopFactory},
// 				},
// 				PrivateKey: pk,
// 				Swarms: map[string]p2p.SecureSwarm{
// 					"mocksec+udp": ssw,
// 				},
// 			})
// 		}
// 		for i := range xs {
// 			for j := range ps {
// 				if i != j {
// 					ps[j].
// 				}
// 			}
// 		}
// 		return xs
// 	})
// }
