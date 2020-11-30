package inet256

func OneHopFactory(params NetworkParams) Network {
	return newSwarmAdapter(params.Swarm, params.Peers)
}
